package resources

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/chanzuckerberg/terraform-provider-snowflake/pkg/snowflake"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/jmoiron/sqlx"
)

func RoleGrants() *schema.Resource {
	return &schema.Resource{
		Create: CreateRoleGrants,
		Read:   ReadRoleGrants,
		Delete: DeleteRoleGrants,
		Update: UpdateRoleGrants,

		Schema: map[string]*schema.Schema{
			"role_name": &schema.Schema{
				Type:        schema.TypeString,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Required:    true,
				Description: "The name of the role we are granting.",
				ValidateFunc: func(val interface{}, key string) ([]string, []error) {
					return snowflake.ValidateIdentifier(val)
				},
			},
			"roles": &schema.Schema{
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
				Description: "Grants role to this specified role.",
			},
			"users": &schema.Schema{
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
				Description: "Grants role to this specified user.",
			},
		},

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func CreateRoleGrants(data *schema.ResourceData, meta interface{}) error {
	db := meta.(*sql.DB)
	roleName := data.Get("role_name").(string)
	roles := expandStringList(data.Get("roles").([]interface{}))
	users := expandStringList(data.Get("users").([]interface{}))

	if len(roles) == 0 && len(users) == 0 {
		return fmt.Errorf("no users or roles specified for role grants")
	}

	for _, role := range roles {
		err := grantRoleToRole(db, roleName, role)
		if err != nil {
			return err
		}
	}

	for _, user := range users {
		err := grantRoleToUser(db, roleName, user)
		if err != nil {
			return err
		}
	}
	data.SetId(roleName)
	return ReadRoleGrants(data, meta)
}

func grantRoleToRole(db *sql.DB, role1, role2 string) error {
	g := snowflake.RoleGrant(role1)
	err := DBExec(db, g.Role(role2).Grant())
	return err
}

func grantRoleToUser(db *sql.DB, role1, user string) error {
	g := snowflake.RoleGrant(role1)
	err := DBExec(db, g.User(user).Grant())
	return err
}

type roleGrant struct {
	CreatedOn   sql.RawBytes   `db:"created_on"`
	Role        sql.NullString `db:"role"`
	GrantedTo   sql.NullString `db:"granted_to"`
	GranteeName sql.NullString `db:"grantee_name"`
	Grantedby   sql.NullString `db:"granted_by"`
}

func ReadRoleGrants(data *schema.ResourceData, meta interface{}) error {
	db := meta.(*sql.DB)
	roleName := data.Id()

	roles := make([]string, 0)
	users := make([]string, 0)

	grants, err := readGrants(db, roleName)
	if err != nil {
		return err
	}

	for _, grant := range grants {
		switch grant.GrantedTo.String {
		case "ROLE":
			roles = append(roles, grant.GranteeName.String)
		case "USER":
			users = append(users, grant.GranteeName.String)
		default:
			return fmt.Errorf("unknown grant type %s", grant.GrantedTo.String)
		}
	}

	err = data.Set("role_name", roleName)
	if err != nil {
		return err
	}
	err = data.Set("roles", roles)
	if err != nil {
		return err
	}
	err = data.Set("users", users)
	if err != nil {
		return err
	}

	return nil
}

func readGrants(db *sql.DB, roleName string) ([]*roleGrant, error) {
	sdb := sqlx.NewDb(db, "snowflake")

	stmt := fmt.Sprintf(`SHOW GRANTS OF ROLE "%s"`, roleName)
	rows, err := sdb.Queryx(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	grants := make([]*roleGrant, 0)
	for rows.Next() {
		g := &roleGrant{}
		err = rows.StructScan(g)
		if err != nil {
			return nil, err
		}
		grants = append(grants, g)

	}

	for _, g := range grants {
		if g.GranteeName.Valid {
			s := g.GranteeName.String
			s = strings.TrimPrefix(s, `"`)
			s = strings.TrimSuffix(s, `"`)
			g.GranteeName = sql.NullString{String: s}
		}
	}

	return grants, nil
}

func DeleteRoleGrants(data *schema.ResourceData, meta interface{}) error {
	db := meta.(*sql.DB)
	roleName := data.Get("role_name").(string)

	roles := expandStringList(data.Get("roles").([]interface{}))
	users := expandStringList(data.Get("users").([]interface{}))

	for _, role := range roles {
		err := revokeRoleFromRole(db, roleName, role)
		if err != nil {
			return err
		}
	}

	for _, user := range users {
		err := revokeRoleFromUser(db, roleName, user)
		if err != nil {
			return err
		}
	}

	data.SetId("")
	return nil
}

func revokeRoleFromRole(db *sql.DB, role1, role2 string) error {
	rg := snowflake.RoleGrant(role1).Role(role2)
	err := DBExec(db, rg.Revoke())
	return err
}

func revokeRoleFromUser(db *sql.DB, role1, user string) error {
	rg := snowflake.RoleGrant(role1).User(user)
	err := DBExec(db, rg.Revoke())
	return err
}

func UpdateRoleGrants(data *schema.ResourceData, meta interface{}) error {
	db := meta.(*sql.DB)
	roleName := data.Get("role_name").(string)

	existingGrants, err := readGrants(db, roleName)
	if err != nil {
		return err
	}

	existingRoleGrants := schema.NewSet(schema.HashString, nil)
	existingUserGrants := schema.NewSet(schema.HashString, nil)

	for _, existingGrant := range existingGrants {
		switch grantedTo := strings.ToUpper(existingGrant.GrantedTo.String); grantedTo {
		case "USER":
			existingUserGrants.Add(existingGrant.GranteeName.String)
		case "ROLE":
			existingRoleGrants.Add(existingGrant.GranteeName.String)
		default:
			return fmt.Errorf("Role granted_to unrecognized type(%s)", grantedTo)
		}
	}

	type funcRoleGrant func(db *sql.DB, resource string, target string) error
	x := func(resource string, grant funcRoleGrant, revoke funcRoleGrant, existingDataSet *schema.Set) error {
		iNewData := data.Get(resource)
		log.Printf("[DEBUG] iNewData %v", iNewData)
		newData := schema.NewSet(schema.HashString, iNewData.([]interface{}))
		log.Printf("[DEBUG] newData %v", newData.List())

		remove := expandStringList(existingDataSet.Difference(newData).List())
		add := expandStringList(newData.Difference(existingDataSet).List())

		for _, user := range remove {
			err := revoke(db, roleName, user)
			if err != nil {
				return err
			}
		}
		for _, user := range add {
			err := grant(db, roleName, user)
			if err != nil {
				return err
			}
		}
		return nil
	}

	err = x("users", grantRoleToUser, revokeRoleFromUser, existingUserGrants)
	if err != nil {
		return err
	}

	err = x("roles", grantRoleToRole, revokeRoleFromRole, existingRoleGrants)
	if err != nil {
		return err
	}

	return ReadRoleGrants(data, meta)
}
