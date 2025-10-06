package tools

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/vintcessun/XMU-CM-Bot/utils"
)

type CourseDepartment struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type CourseInstructor struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type CourseData struct {
	Department  CourseDepartment    `json:"department"`
	Id          int                 `json:"id"`
	Instructors []*CourseInstructor `json:"instructors"`
	Name        string              `json:"name"`
	StartDate   string              `json:"start_date"`
	CourseCode  string              `json:"course_code"`
}

type APICourseData struct {
	Courses []*CourseData `json:"courses"`
}

type APIRecentCourseData struct {
	VisitedCourses []*CourseData `json:"visited_courses"`
}

type FormatCourseData []*FormatCourseInside

type FormatCourseInside struct {
	Id         int    `json:"id"`
	Name       string `json:"name"`
	Department string `json:"department"`
	Semester   string `json:"semester"`
}

func (f *FormatCourseData) ToString() (string, error) {
	return utils.MarshalJSON[FormatCourseData](f)
}

func (f *FormatCourseData) Get(id int) (*FormatCourseInside, bool) {
	for _, v := range *f {
		if v.Id == id {
			return v, true
		}
	}
	return nil, false
}

func GetCourseData(client *resty.Client) (*FormatCourseData, error) {
	var data FormatCourseData
	res, err := client.R().Get("https://lnt.xmu.edu.cn/api/my-courses?showScorePassedStatus=false")
	if err != nil {
		return &data, err
	}
	unformatData, err := utils.UnmarshalJSON[APICourseData](res.Body())
	if err != nil {
		return &data, err
	}
	for _, course := range unformatData.Courses {
		new_data := FormatCourseInside{Id: course.Id, Name: course.Name, Department: course.Department.Name}
		seme, err := GetSemesterStr(course.StartDate)
		if err != nil {
			Logger.Warning("获得课程学期失败 ", err)
			Logger.Info("课程信息 ", course)
		} else {
			new_data.Semester = seme
		}
		data = append(data, &new_data)
	}

	return &data, nil
}

func GetSemesterStr(dateStr string) (string, error) {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return "", fmt.Errorf("日期格式错误，请使用 YYYY-MM-DD: %w", err)
	}
	return GetSemesterInfo(t), nil
}

func GetSemesterInfo(t time.Time) string {
	year := t.Year()
	month := int(t.Month())

	var academicYearStart, academicYearEnd int
	var semester string

	// 根据月份判断学期和学年
	switch {
	case 6 <= month && month <= 7:
		// 夏季学期（6-7月）
		academicYearStart = year - 1
		academicYearEnd = year
		semester = "第三学期（小学期，夏季学期）"
	case month >= 8 || month == 1:
		// 秋季学期（8-1月）
		if month >= 8 {
			academicYearStart = year
		} else { // month == 1
			academicYearStart = year - 1
		}
		academicYearEnd = academicYearStart + 1
		semester = "第一学期（上学期，秋季学期）"
	default:
		// 春季学期（2-5月）
		academicYearStart = year - 1
		academicYearEnd = year
		semester = "第二学期（下学期，春季学期）"
	}

	// 格式化返回字符串
	return fmt.Sprintf("%d-%d学年 %s", academicYearStart, academicYearEnd, semester)
}

func GetRecentCourseData(client *resty.Client) (*FormatCourseData, error) {
	var data FormatCourseData
	res, err := client.R().Get("https://lnt.xmu.edu.cn/api/user/recently-visited-courses")
	if err != nil {
		return &data, nil
	}
	unformatData, err := utils.UnmarshalJSON[APIRecentCourseData](res.Body())
	for _, course := range unformatData.VisitedCourses {
		new_data := FormatCourseInside{Id: course.Id, Name: course.Name, Department: course.Department.Name}
		data = append(data, &new_data)
	}
	if err != nil {
		return &data, err
	}
	return &data, nil
}

type LLMCourseResponse struct {
	Course *int `json:"course"`
}

func getLLMChoosePrompt(courseData, recentCourseData *FormatCourseData, command string) string {
	date := time.Now()
	formatData := date.Format("2006-01-02")
	semester := GetSemesterInfo(date)
	courseDataFormat, err := courseData.ToString()
	if err != nil {
		courseDataFormat = ""
	}
	recentCourseDataFormat, err := recentCourseData.ToString()
	if err != nil {
		recentCourseDataFormat = ""
	}
	data := []string{`你是一个专业的理解用户需求的客服，请根据用户的需求字符串和现有信息推测用户最可能选择的信息并且按照要求返回JSON
===
# 返回的要求
键为“course”(str类型)，值为int类型的
一定要符合这个格式：{"course":course_id}
## 示例 - 找到课程
{"course":62239}
## 示例 - 没找到课程
{"course":null}
## 注意事项
1.  除了回复使用的工具之外，不要使用任何其他文字进行修饰，保证输出的全部为 JSON ！！！
2.  一定要按照要求返回指定的格式，请严格遵照要求！！！不要出现返回{}空JSON的形式
===
# 一些基本的信息
## 对于传入参数的解释
传入的课程参数为一个list[dict]类型的参数，list中每个元素代表一个课程。
### 对于每个课程(dict类型)的参数的解释
#### id
类型: int
解释: 课程号，如果选中这门课程返回的就是这个id，即为course_id。
#### name
类型: str
解释: 课程的名称，用这个来主要的筛选用户的需求。
#### department
类型: str
解释: 开课单位，如果用户有提到可以用这个筛选。
#### semester
类型: str
解释: 代表课程所处的学期，如果用户没有提到筛选的学期默认为最近的1-2个学期在筛选的范围内。
## 对于学期的大致定义
在每年的08-01到01-31，如“2024-09-02”为“2024-2025学年 第一学期（上学期，秋季学期）”
在每年的02-01到05-31，如“2025-02-17”为“2024-2025学年 第二学期（下学期，春季学期）”
在每年的06-01到07-31，如“2025-06-20”为“2024-2025学年 第三学期（小学期，夏季学期）”
===
# 传入的课程参数：
## 所有课程的数据
`, courseDataFormat, `
## 最近访问的课程的数据
`, recentCourseDataFormat, `
===
# 一些其他参数
## 时间参数
### 当前日期
`, formatData, `
### 当前大致学期
`, semester, `
===
用户的请求：`, command,
	}
	return strings.Join(data, "")
}

func GetCourseById(id int, client *resty.Client) (*FormatCourseInside, error) {
	res, err := client.R().Get(fmt.Sprintf("https://lnt.xmu.edu.cn/api/courses/%d?fields=id,name,department,start_date", id))
	if err != nil {
		return nil, err
	}
	course, err := utils.UnmarshalJSON[CourseData](res.Body())
	if err != nil {
		return nil, err
	}
	data := FormatCourseInside{Id: course.Id, Name: course.Name, Department: course.Department.Name}
	seme, err := GetSemesterStr(course.StartDate)
	if err != nil {
		Logger.Warning("获得课程学期失败 ", err)
		Logger.Info("课程信息 ", course)
	} else {
		data.Semester = seme
	}
	return &data, nil
}

func GetLLMChooseCourse(courseData, recentCourseData *FormatCourseData, command string, client *resty.Client) (*FormatCourseInside, error) {
	prompt := getLLMChoosePrompt(courseData, recentCourseData, command)
	msg := LoopGetJsonReturn[LLMCourseResponse](Llm.Choice, prompt)
	if msg.Course == nil {
		return nil, errors.New("请更加清晰阐明是哪一门课")
	}
	courseId := *msg.Course
	Logger.Info("获取到课程id: ", courseId)
	course, ok := courseData.Get(courseId)
	if !ok {
		var err error
		course, err = GetCourseById(courseId, client)
		if err != nil {
			return nil, err
		}
	}
	Logger.Info("课程名为: ", course.Name)
	return course, nil
}

type CourseActivityUpload struct {
	Name        string `json:"name"`
	ReferenceId int    `json:"reference_id"`
}

type CourseActivity struct {
	Title   string                 `json:"title"`
	Uploads []CourseActivityUpload `json:"uploads"`
}

type APICourseActivities struct {
	Activities []CourseActivity `json:"activities"`
}

type FormatFileData = []*FormatFileInside

type FormatFileInside struct {
	Name string `json:"name"`
	Id   int    `json:"id"`
}

func GetCourseActivities(courseId int, client *resty.Client) (*FormatFileData, error) {
	var data FormatFileData
	res, err := client.R().Get(fmt.Sprintf("https://lnt.xmu.edu.cn/api/courses/%d/activities", courseId))
	if err != nil {
		return &data, err
	}

	unformatData, err := utils.UnmarshalJSON[APICourseActivities](res.Body())
	if err != nil {
		return &data, err
	}

	for _, activity := range unformatData.Activities {
		title := activity.Title
		for _, upload := range activity.Uploads {
			data = append(data, &FormatFileInside{Name: strings.Join([]string{title, upload.Name}, "-"), Id: upload.ReferenceId})
		}
	}

	return &data, nil
}

type ApiFileReferenceIdUrl struct {
	Url string `json:"url"`
}

func GetURLById(file *FormatFileInside, client *resty.Client) (string, error) {
	res, err := client.R().Get(fmt.Sprintf("https://lnt.xmu.edu.cn/api/uploads/reference/%d/url", file.Id))
	if err != nil {
		return "", err
	}
	body := res.Body()
	apiResponse, err := utils.UnmarshalJSON[ApiFileReferenceIdUrl](body)
	if err != nil {
		return "", err
	}
	return apiResponse.Url, nil
}

type CourseCache struct {
	m sync.Map
}

func (p *CourseCache) get(key int) (*CourseData, bool) {
	data, ok := p.m.Load(key)
	if !ok {
		return nil, ok
	} else {
		switch e := data.(type) {
		case CourseData:
			return &e, ok
		case *CourseData:
			return e, ok
		default:
			return nil, false
		}
	}
}

func (p *CourseCache) insert(key int, value *CourseData) {
	p.m.Store(key, value)
}

var CourseCacheValue CourseCache

func GetCourseInfo(client *resty.Client, courseId int) (*CourseData, error) {
	ret, ok := CourseCacheValue.get(courseId)
	if ok {
		return ret, nil
	}

	resp, err := client.R().Get(fmt.Sprintf("https://lnt.xmu.edu.cn/api/courses/%d?fields=name,course_code,instructors(name)", courseId))
	if err != nil {
		return nil, err
	}
	body := resp.Body()
	data, err := utils.UnmarshalJSON[CourseData](body)
	if err != nil {
		return nil, err
	}

	CourseCacheValue.insert(courseId, data)

	return data, nil
}
