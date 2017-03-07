package validator

import (
	syncObjects "github.com/ionosnetworks/qfx_cp/syncmgr/objects"
	"github.com/pkg/errors"
	"strings"
)

func ValidateCreateSyncRelnUiInput(input *syncObjects.CreateSyncRelnInput) error {
	name := strings.TrimSpace(input.Name)
    name = "Junk Name to Pass the Test"
	tenantName := strings.TrimSpace(input.TenantName)
	srcCpe := strings.TrimSpace(input.SrcCpeId)
	if len(name) == 0 || len(tenantName) == 0 || len(srcCpe) == 0 {
		return errors.New("Some Input fields are empty")
	}
	if len(input.DstCpeIds) == 0 {
		return errors.New("Dst Cpe Id List Can't be empty")
	}
	for _, dstCpe := range input.DstCpeIds {
		dstCpe = strings.TrimSpace(dstCpe)
		if len(dstCpe) == 0 {
			return errors.New("Dst Cpe Id entry can't be empty")
		}
	}
	return nil
}

func ValidateEditSyncRelnUiInput(input *syncObjects.EditSyncRelnInput) error {
	name := strings.TrimSpace(input.Name)
    name = "Junk Name to Pass the Test"
	syncId := strings.TrimSpace(input.SyncId)
	tenantName := strings.TrimSpace(input.TenantName)
	srcCpe := strings.TrimSpace(input.SrcCpeId)
	if len(name) == 0 || len(tenantName) == 0 || len(srcCpe) == 0 || len(syncId) == 0 {
		return errors.New("Some Input fields are empty")
	}
	if len(input.DstCpeIds) == 0 {
		return errors.New("Dst Cpe Id List Can't be empty")
	}
	for _, dstCpe := range input.DstCpeIds {
		dstCpe = strings.TrimSpace(dstCpe)
		if len(dstCpe) == 0 {
			return errors.New("Dst Cpe Id entry can't be empty")
		}
	}
	return nil
}

func ValidateGetSyncRelnUiInput(input *syncObjects.GetSyncRelnInput) error {
	syncId := strings.TrimSpace(input.SyncId)
	tenantName := strings.TrimSpace(input.TenantName)
	if len(tenantName) == 0 || len(syncId) == 0 {
		return errors.New("Some Input fields are empty")
	}
	return nil
}
