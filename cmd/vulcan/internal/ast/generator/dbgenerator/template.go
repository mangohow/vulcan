package dbgenerator

var (
	MapperDeclareTemplate = `type {{ .ModelStructName }} struct {
	db *sql.DB
}`

	MapperNewFuncTemplate = `func New{{ .ModelStructName }}(db *sql.DB) *{{ .ModelStructName }} {
	return &{{ .ModelStructName }}{db: db}
}`

	AddFuncTemplate = `func ({{ .ReceiverName }} *{{ .ModelStructName }}) Add({{ .ModelObjName }} *{{ .ModelTypeName }}) {
	Insert("INSERT INTO {{ .TableName }} ({{ .TableFields }}) VALUES ({{ .StructFields }})")
}`

	BatchAddFuncTemplate = `func ({{ ReceiverName }} *{{ ModelStructName }}) BatchAdd({{ .EntityObjName }} []*{{ .ModelPackageName }}.{{ .ModelTypeName }}) {
	Insert(SQL().
		Stmt("INSERT INTO {{.TableName}} ({{.TableFields}}) VALUES").
		Foreach("{{ .EntityObjName }}", "{{ EntityObjNameSingle }}", "", "", "", "").
		Build())
	return nil
}`

	GetByIdFuncTemplate = `func ({{ ReceiverName }} *{{ ModelStructName }}) GetById({{ .QueryKeyName }} {{ .QueryKeyType }}) *{{ .ModelPackageName }}.{{ .ModelTypeName }} {
	Select("SELECT * FROM {{ .TableName }} WHERE {{ .PrimaryKeyName }} = {{ .QueryKeyNameBracket }}")
	return nil
}`

	SelectListByIdsFuncTemplate = `func ({{ ReceiverName }} *{{ ModelStructName }}) SelectListByIds({{ .QueryKeyName }} []{{ .QueryKeyType }}) []*{{ .ModelPackageName }}.{{ .ModelTypeName }} {
	Select(SQL().
		Stmt("SELECT * FROM {{ .TableName }} WHERE {{ PrimaryKeyName }} IN").
		Foreach("{{ .QueryKeyName }}", "{{ .QueryKeyNameSingle }}", ", ", "(", ")", "{{ .QueryKeyNameSingleBracket }}").
		Build())
	return nil
}`

	SelectPageFuncTemplate = `func ({{ ReceiverName }} *{{ ModelStructName }}) SelectPage(page vulcan.Page) []*model.User {
	Select("SELECT * FROM {{ .TableName }}")
	return nil
}`
	UpdateByIdFuncTemplate = `func ({{ ReceiverName }} *{{ ModelStructName }}) UpdateById({{ .QueryKeyName }} {{ .QueryKeyType }}) {
    Update(UPDATE {{ .TableName }} SET {{ .SetFields }})
}`
)

type MapperTemplateOptions struct {
	MapperName string
}

type AddFuncTemplateOptions struct {
	ReceiverName  string // 接收器变量名
	MapperName    string // mapper结构体名
	ModelObjName  string // 结构体模型名
	ModelTypeName string // 结构体模型类型名, 如果跟mapper不在同一个包, 则携带包名
	TableName     string // 数据库表名
	TableFields   string // 表所有字段
	StructFields  string // 所有结构体字段
}
