package helpers

import (
	_ "embed"
	"strings"

	"gorm.io/gorm"
)

//go:embed sql/000003_create_tenant_schema.up.sql
var tenantSchemaUpSQL string

//go:embed sql/000003_create_tenant_schema.down.sql
var tenantSchemaDownSQL string

func CreateTenantSchema(db *gorm.DB, schemaName string) error {
	sql := strings.ReplaceAll(tenantSchemaUpSQL, ":schema_name", schemaName)
	return db.Exec(sql).Error
}

func DropTenantSchema(db *gorm.DB, schemaName string) error {
	sql := strings.ReplaceAll(tenantSchemaDownSQL, ":schema_name", schemaName)
	return db.Exec(sql).Error
}
