package model

// ExportFile 导出文件内容
type ExportFile struct {
	FileName    string
	ContentType string
	Content     []byte
}

type VolunteerExportRow struct {
	VolunteerID  int64   `excel:"志愿者ID" csv:"志愿者ID"`
	RealName     string  `excel:"姓名" csv:"姓名"`
	Gender       string  `excel:"性别" csv:"性别"`
	Mobile       string  `excel:"手机号" csv:"手机号"`
	Email        string  `excel:"邮箱" csv:"邮箱"`
	Organization string  `excel:"所属组织" csv:"所属组织"`
	TotalHours   float64 `excel:"累计工时" csv:"累计工时"`
	ServiceCount int32   `excel:"服务次数" csv:"服务次数"`
	Status       string  `excel:"志愿者状态" csv:"志愿者状态"`
	AuditStatus  string  `excel:"实名状态" csv:"实名状态"`
	CreatedAt    string  `excel:"创建时间" csv:"创建时间"`
}

type ActivityExportRow struct {
	ActivityID    int64   `excel:"活动ID" csv:"活动ID"`
	Title         string  `excel:"标题" csv:"标题"`
	Description   string  `excel:"描述" csv:"描述"`
	StartTime     string  `excel:"开始时间" csv:"开始时间"`
	EndTime       string  `excel:"结束时间" csv:"结束时间"`
	Location      string  `excel:"地点" csv:"地点"`
	Address       string  `excel:"地址" csv:"地址"`
	Duration      float64 `excel:"预估工时" csv:"预估工时"`
	MaxPeople     int32   `excel:"最大人数" csv:"最大人数"`
	CurrentPeople int32   `excel:"当前人数" csv:"当前人数"`
	Status        string  `excel:"活动状态" csv:"活动状态"`
	Organization  string  `excel:"发布组织" csv:"发布组织"`
	CreatedAt     string  `excel:"创建时间" csv:"创建时间"`
}

type OpsReportExportRow struct {
	PeriodType      string `excel:"Period Type" csv:"Period Type"`
	OrganizationID  int64  `excel:"Organization ID" csv:"Organization ID"`
	Start           string `excel:"Start" csv:"Start"`
	End             string `excel:"End" csv:"End"`
	ActivitiesCount int64  `excel:"Activities Count" csv:"Activities Count"`
	SignupsCount    int64  `excel:"Signups Count" csv:"Signups Count"`
	AttendanceCount int64  `excel:"Attendance Count" csv:"Attendance Count"`
	WorkhoursCount  int64  `excel:"Workhours Count" csv:"Workhours Count"`
}
