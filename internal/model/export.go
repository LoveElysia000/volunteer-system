package model

// ExportFile 导出文件内容
type ExportFile struct {
	FileName    string
	ContentType string
	Content     []byte
}

type VolunteerExportRow struct {
	VolunteerID  int64   `excel:"志愿者ID"`
	RealName     string  `excel:"姓名"`
	Gender       string  `excel:"性别"`
	Mobile       string  `excel:"手机号"`
	Email        string  `excel:"邮箱"`
	Organization string  `excel:"所属组织"`
	TotalHours   float64 `excel:"累计工时"`
	ServiceCount int32   `excel:"服务次数"`
	Status       string  `excel:"志愿者状态"`
	AuditStatus  string  `excel:"实名状态"`
	CreatedAt    string  `excel:"创建时间"`
}

type ActivityExportRow struct {
	ActivityID    int64   `excel:"活动ID"`
	Title         string  `excel:"标题"`
	Description   string  `excel:"描述"`
	StartTime     string  `excel:"开始时间"`
	EndTime       string  `excel:"结束时间"`
	Location      string  `excel:"地点"`
	Address       string  `excel:"地址"`
	Duration      float64 `excel:"预估工时"`
	MaxPeople     int32   `excel:"最大人数"`
	CurrentPeople int32   `excel:"当前人数"`
	Status        string  `excel:"活动状态"`
	Organization  string  `excel:"发布组织"`
	CreatedAt     string  `excel:"创建时间"`
}
