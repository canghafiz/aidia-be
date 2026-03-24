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

func normalizeSchema(username string) string {
	schema := strings.ToLower(username)
	schema = strings.ReplaceAll(schema, " ", "_")
	schema = strings.ReplaceAll(schema, "-", "_")
	return schema
}

func NormalizeSchema(username string) string {
	return normalizeSchema(username)
}

func CreateTenantSchema(db *gorm.DB, schemaName string) error {
	schemaName = normalizeSchema(schemaName)
	sql := strings.ReplaceAll(tenantSchemaUpSQL, ":schema_name", schemaName)

	statements := splitStatements(sql)
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func DropTenantSchema(db *gorm.DB, schemaName string) error {
	schemaName = normalizeSchema(schemaName)
	sql := strings.ReplaceAll(tenantSchemaDownSQL, ":schema_name", schemaName)

	statements := splitStatements(sql)
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func splitStatements(sql string) []string {
	var statements []string
	var current strings.Builder
	inDollarQuote := false

	lines := strings.Split(sql, "\n")
	for _, line := range lines {
		if strings.Contains(line, "$$") {
			inDollarQuote = !inDollarQuote
		}
		current.WriteString(line + "\n")
		if !inDollarQuote && strings.HasSuffix(strings.TrimSpace(line), ";") {
			statements = append(statements, current.String())
			current.Reset()
		}
	}

	return statements
}
