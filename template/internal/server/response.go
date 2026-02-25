package server

import (
	"encoding/json"
	stdhttp "net/http"
	"reflect"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/encoding"
	kratosjson "github.com/go-kratos/kratos/v2/encoding/json"
	kratoserrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"

	"spark/internal/common/errors"
	"spark/internal/i18n"
	"spark/pkg/trace"
)

const (
	baseContentType = "application"
)

// DefaultResponseEncoder 标准响应编码器
func DefaultResponseEncoder(w http.ResponseWriter, r *stdhttp.Request, v interface{}) error {
	if v == nil {
		return nil
	}
	if rd, ok := v.(http.Redirector); ok {
		url, code := rd.Redirect()
		stdhttp.Redirect(w, r, url, code)
		return nil
	}

	// 从 response header 中获取 trace ID（由 TraceHeader 中间件设置）
	traceID := w.Header().Get("X-Trace-ID")

	// 创建响应的基本结构
	result := map[string]interface{}{
		"code":       200,
		"msg":        "操作成功",
		"timestamp":  getCurrentTimestamp(),
		"request_id": traceID,
	}

	// 使用反射检查结构体的有效字段数量
	validFieldCount := countValidFields(v)

	codec := encoding.GetCodec(kratosjson.Name)
	data, err := codec.Marshal(v)
	if err != nil {
		return err
	}

	// 解析原始数据
	var responseData map[string]interface{}
	if err = json.Unmarshal(data, &responseData); err == nil {
		// 根据反射得到的有效字段数量决定是否打平
		if validFieldCount == 1 {
			// 如果只有一个有效字段，则直接使用其值作为data
			for _, v := range responseData {
				result["data"] = v
				break
			}
		} else {
			// 如果有多个有效字段或没有字段，使用整个responseData作为data
			result["data"] = responseData
		}
	}

	w.Header().Set("Content-Type", ContentType(codec.Name()))
	bs, err := json.Marshal(result)
	if err != nil {
		return err
	}

	_, err = w.Write(bs)
	return err
}

// DefaultErrorEncoder 标准错误编码器（支持 i18n 翻译）
func DefaultErrorEncoder(w http.ResponseWriter, r *stdhttp.Request, err error) {
	ctx := r.Context()

	// 尝试从多个来源获取 trace ID
	// 1. 首先尝试从 Kratos transport 的 ReplyHeader 获取（由 TraceHeader 中间件设置）
	var traceID string
	if tr, ok := transport.FromServerContext(ctx); ok {
		if header := tr.ReplyHeader(); header != nil {
			traceID = header.Get("X-Trace-ID")
		}
	}

	// 2. 如果没有，从 context 获取
	if traceID == "" {
		traceID = trace.TraceIDFromContext(ctx)
	}

	requestID := traceID

	e := kratoserrors.FromError(err)

	// 检查是否是可翻译错误
	msg := e.Message
	if transErr, ok := errors.IsTranslatableError(err); ok {
		// 使用 i18n 翻译错误消息
		msg = i18n.T(ctx, transErr.Reason, transErr.TemplateData)
		// 使用可翻译错误中指定的 HTTP 状态码
		if transErr.HTTPCode > 0 {
			e.Code = transErr.HTTPCode
		}
	}

	// 添加 reason 字段用于前端错误识别
	result := map[string]interface{}{
		"code":       e.Code,
		"reason":     e.Reason,
		"msg":        msg,
		"data":       nil,
		"timestamp":  getCurrentTimestamp(),
		"request_id": requestID,
	}

	// Set headers
	if traceID != "" {
		w.Header().Set("X-Trace-ID", traceID)
	}
	w.Header().Set("Content-Type", ContentType(kratosjson.Name))

	codec := encoding.GetCodec(kratosjson.Name)
	body, err := codec.Marshal(result)
	if err != nil {
		w.WriteHeader(stdhttp.StatusInternalServerError)
		return
	}
	w.WriteHeader(stdhttp.StatusOK) // 始终返回200，错误信息在body中
	_, _ = w.Write(body)
}

// countValidFields 计算结构体中有json或protobuf tag的字段数量
func countValidFields(v interface{}) int {
	if v == nil {
		return 0
	}

	rv := reflect.ValueOf(v)
	// 如果是指针，获取其指向的值
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return 0
		}
		rv = rv.Elem()
	}

	// 只处理结构体
	if rv.Kind() != reflect.Struct {
		return 1 // 非结构体类型认为是一个字段
	}

	count := 0
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)

		// 跳过未导出的字段
		if !field.IsExported() {
			continue
		}

		// 检查字段是否有json或protobuf tag
		jsonTag := field.Tag.Get("json")
		protobufTag := field.Tag.Get("protobuf")

		if jsonTag != "" || protobufTag != "" {
			count++
		}
	}

	return count
}

// ContentType returns the content-type with base prefix.
func ContentType(subtype string) string {
	return strings.Join([]string{baseContentType, subtype}, "/")
}

// getCurrentTimestamp returns current timestamp in ISO8601 format
func getCurrentTimestamp() string {
	return time.Now().Format(time.RFC3339)
}
