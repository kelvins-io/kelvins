package gin_helper

var MsgFlags = map[int]string{
	SUCCESS:           "ok",
	ERROR:             "服务器出错",
	Unknown:           "未知",
	TooManyRequests:   "请求太多，稍后再试",
	InvalidParams:     "请求参数错误",
	ErrorTokenEmpty:   "用户token为空",
	ErrorTokenInvalid: "用户token无效",
	ErrorTokenExpire:  "用户token过期",
	ErrorUserNotExist: "用户不存在",
}

func GetMsg(code int) string {
	msg, ok := MsgFlags[code]
	if ok {
		return msg
	}
	return MsgFlags[Unknown]
}
