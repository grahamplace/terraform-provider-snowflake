package snowflake_test

import (
	"testing"

	"github.com/chanzuckerberg/terraform-provider-snowflake/pkg/snowflake"
	"github.com/stretchr/testify/require"
)

func TestNetworkPolicyDrop(t *testing.T) {
	r := require.New(t)
	s := snowflake.NetworkPolicy("test_network_policy")
	r.NotNil(s)

	q := s.Drop()
	r.Equal(`DROP NETWORK POLICY "test_network_policy"`, q)
}

func TestNetworkPolicyChangeComment(t *testing.T) {
	r := require.New(t)
	s := snowflake.NetworkPolicy("test_network_policy")
	r.NotNil(s)

	q := s.ChangeComment("test comment!")
	r.Equal(`ALTER NETWORK POLICY "test_network_policy" SET COMMENT = 'test comment!'`, q)
}

func TestNetworkPolicyRemoveComment(t *testing.T) {
	r := require.New(t)
	s := snowflake.NetworkPolicy("test_network_policy")
	r.NotNil(s)

	q := s.RemoveComment()
	r.Equal(`ALTER NETWORK POLICY "test_network_policy" UNSET COMMENT`, q)
}

func TestNetworkPolicyChangeIpList(t *testing.T) {
	r := require.New(t)
	s := snowflake.NetworkPolicy("test_network_policy")
	r.NotNil(s)

	newAllowedIps := []string{"192.168.0.100/24", "29.254.123.20"}
	q := s.ChangeIpList("ALLOWED", newAllowedIps)
	r.Equal(`ALTER NETWORK POLICY "test_network_policy" SET ALLOWED_IP_LIST = ('192.168.0.100/24', '29.254.123.20')`, q)

	var newBlockedIps []string
	q = s.ChangeIpList("BLOCKED", newBlockedIps)
	r.Equal(`ALTER NETWORK POLICY "test_network_policy" SET BLOCKED_IP_LIST = ()`, q)
}

func TestNetworkPolicySetOnAccount(t *testing.T) {
	r := require.New(t)
	s := snowflake.NetworkPolicy("test_network_policy")
	r.NotNil(s)

	q := s.SetOnAccount()
	r.Equal(`ALTER ACCOUNT SET NETWORK_POLICY = "test_network_policy"`, q)
}

func TestNetworkPolicyUnsetOnAccount(t *testing.T) {
	r := require.New(t)
	s := snowflake.NetworkPolicy("test_network_policy")
	r.NotNil(s)

	q := s.UnsetOnAccount()
	r.Equal(`ALTER ACCOUNT UNSET NETWORK_POLICY`, q)
}

func TestNetworkPolicySetOnUser(t *testing.T) {
	r := require.New(t)
	s := snowflake.NetworkPolicy("test_network_policy")
	r.NotNil(s)

	q := s.SetOnUser("testuser")
	r.Equal(`ALTER USER "testuser" SET NETWORK_POLICY = "test_network_policy"`, q)
}

func TestNetworkPolicyUnsetOnUser(t *testing.T) {
	r := require.New(t)
	s := snowflake.NetworkPolicy("test_network_policy")
	r.NotNil(s)

	q := s.UnsetOnUser("testuser")
	r.Equal(`ALTER USER "testuser" UNSET NETWORK_POLICY`, q)
}
