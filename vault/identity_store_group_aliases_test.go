package vault

import (
	"testing"

	"github.com/hashicorp/vault/helper/identity"
	"github.com/hashicorp/vault/helper/namespace"
	"github.com/hashicorp/vault/logical"
)

func TestIdentityStore_GroupAliasDeletionOnGroupDeletion(t *testing.T) {
	var resp *logical.Response
	var err error

	ctx := namespace.RootContext(nil)
	i, accessor, _ := testIdentityStoreWithGithubAuth(ctx, t)

	resp, err = i.HandleRequest(ctx, &logical.Request{
		Path:      "group",
		Operation: logical.UpdateOperation,
		Data: map[string]interface{}{
			"type": "external",
		},
	})
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("bad: resp: %#v\nerr: %v\n", resp, err)
	}
	groupID := resp.Data["id"].(string)

	resp, err = i.HandleRequest(ctx, &logical.Request{
		Path:      "group-alias",
		Operation: logical.UpdateOperation,
		Data: map[string]interface{}{
			"name":           "testgroupalias",
			"mount_accessor": accessor,
			"canonical_id":   groupID,
		},
	})
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("bad: resp: %#v\nerr: %v\n", resp, err)
	}
	groupAliasID := resp.Data["id"].(string)

	resp, err = i.HandleRequest(ctx, &logical.Request{
		Path:      "group/id/" + groupID,
		Operation: logical.DeleteOperation,
	})
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("bad: resp: %#v\nerr: %v", resp, err)
	}

	resp, err = i.HandleRequest(ctx, &logical.Request{
		Path:      "group-alias/id/" + groupAliasID,
		Operation: logical.ReadOperation,
	})
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("bad: resp: %#v\nerr: %v", resp, err)
	}
	if resp != nil {
		t.Fatalf("expected a nil response")
	}
}

func TestIdentityStore_GroupAliases_CRUD(t *testing.T) {
	var resp *logical.Response
	var err error
	ctx := namespace.RootContext(nil)
	i, accessor, _ := testIdentityStoreWithGithubAuth(ctx, t)

	groupReq := &logical.Request{
		Path:      "group",
		Operation: logical.UpdateOperation,
		Data: map[string]interface{}{
			"type": "external",
		},
	}
	resp, err = i.HandleRequest(ctx, groupReq)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("bad: resp: %#v\nerr: %v\n", resp, err)
	}
	groupID := resp.Data["id"].(string)

	groupAliasReq := &logical.Request{
		Path:      "group-alias",
		Operation: logical.UpdateOperation,
		Data: map[string]interface{}{
			"name":           "testgroupalias",
			"mount_accessor": accessor,
			"canonical_id":   groupID,
			"mount_type":     "ldap",
		},
	}
	resp, err = i.HandleRequest(ctx, groupAliasReq)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("bad: resp: %#v\nerr: %v\n", resp, err)
	}
	groupAliasID := resp.Data["id"].(string)

	groupAliasReq.Path = "group-alias/id/" + groupAliasID
	groupAliasReq.Operation = logical.ReadOperation
	resp, err = i.HandleRequest(ctx, groupAliasReq)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("bad: resp: %#v\nerr: %v\n", resp, err)
	}

	if resp.Data["id"].(string) != groupAliasID {
		t.Fatalf("bad: group alias: %#v\n", resp.Data)
	}

	resp, err = i.HandleRequest(ctx, &logical.Request{
		Path:      "group-alias/id/" + groupAliasID,
		Operation: logical.UpdateOperation,
		Data: map[string]interface{}{
			"name":           "testupdatedgroupaliasname",
			"mount_accessor": accessor,
			"canonical_id":   groupID,
			"mount_type":     "ldap",
		},
	})
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("bad: err: %v; resp: %#v", err, resp)
	}
	if resp.Data["id"].(string) != groupAliasID {
		t.Fatalf("bad: group alias: %#v\n", resp.Data)
	}

	groupAliasReq.Operation = logical.DeleteOperation
	resp, err = i.HandleRequest(ctx, groupAliasReq)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("bad: resp: %#v\nerr: %v\n", resp, err)
	}

	groupAliasReq.Operation = logical.ReadOperation
	resp, err = i.HandleRequest(ctx, groupAliasReq)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("bad: resp: %#v\nerr: %v\n", resp, err)
	}

	if resp != nil {
		t.Fatalf("failed to delete group alias")
	}
}

func TestIdentityStore_GroupAliases_MemDBIndexes(t *testing.T) {
	var err error
	ctx := namespace.RootContext(nil)
	i, accessor, _ := testIdentityStoreWithGithubAuth(ctx, t)

	group := &identity.Group{
		ID:   "testgroupid",
		Name: "testgroupname",
		Metadata: map[string]string{
			"testmetadatakey1": "testmetadatavalue1",
			"testmetadatakey2": "testmetadatavalue2",
		},
		Alias: &identity.Alias{
			ID:            "testgroupaliasid",
			Name:          "testalias",
			MountAccessor: accessor,
			CanonicalID:   "testgroupid",
			MountType:     "ldap",
		},
		ParentGroupIDs:  []string{"testparentgroupid1", "testparentgroupid2"},
		MemberEntityIDs: []string{"testentityid1", "testentityid2"},
		Policies:        []string{"testpolicy1", "testpolicy2"},
		BucketKeyHash:   i.groupPacker.BucketKeyHashByItemID("testgroupid"),
	}

	txn := i.db.Txn(true)
	defer txn.Abort()
	err = i.MemDBUpsertAliasInTxn(txn, group.Alias, true)
	if err != nil {
		t.Fatal(err)
	}
	err = i.MemDBUpsertGroupInTxn(txn, group)
	if err != nil {
		t.Fatal(err)
	}
	txn.Commit()

	alias, err := i.MemDBAliasByID("testgroupaliasid", false, true)
	if err != nil {
		t.Fatal(err)
	}
	if alias.ID != "testgroupaliasid" {
		t.Fatalf("bad: group alias: %#v\n", alias)
	}

	group, err = i.MemDBGroupByAliasID("testgroupaliasid", false)
	if err != nil {
		t.Fatal(err)
	}
	if group.ID != "testgroupid" {
		t.Fatalf("bad: group: %#v\n", group)
	}

	aliasByFactors, err := i.MemDBAliasByFactors(group.Alias.MountAccessor, group.Alias.Name, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if aliasByFactors.ID != "testgroupaliasid" {
		t.Fatalf("bad: group alias: %#v\n", aliasByFactors)
	}
}

func TestIdentityStore_GroupAliases_AliasOnInternalGroup(t *testing.T) {
	var err error
	var resp *logical.Response

	ctx := namespace.RootContext(nil)
	i, accessor, _ := testIdentityStoreWithGithubAuth(ctx, t)

	groupReq := &logical.Request{
		Path:      "group",
		Operation: logical.UpdateOperation,
	}
	resp, err = i.HandleRequest(ctx, groupReq)
	if err != nil || (resp != nil && resp.IsError()) {
		t.Fatalf("bad: resp: %#v; err: %v", resp, err)
	}
	groupID := resp.Data["id"].(string)

	aliasReq := &logical.Request{
		Path:      "group-alias",
		Operation: logical.UpdateOperation,
		Data: map[string]interface{}{
			"name":           "testname",
			"mount_accessor": accessor,
			"canonical_id":   groupID,
		},
	}
	resp, err = i.HandleRequest(ctx, aliasReq)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.IsError() {
		t.Fatalf("expected an error")
	}
}
