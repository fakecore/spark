package response

type BaseResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type PageQuery struct {
	Current  int32 `form:"current,default=1"`
	PageSize int32 `form:"page_size,default=10"`
}

var (
	SuccessCode = 200
	FailCode    = 500
)

func Success() BaseResp {
	return BaseResp{
		Code: 200,
		Msg:  "操作成功",
	}

}

func SuccessWithMsg() {

}

func Fail() BaseResp {
	return BaseResp{
		Code: 500,
		Msg:  "操作失败",
	}
}

func FailWithMsg(msg string) BaseResp {
	return BaseResp{
		Code: 500,
		Msg:  msg,
	}
}
