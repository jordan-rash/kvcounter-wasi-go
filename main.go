package main

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"strconv"
	"strings"

	kvcounter "kvcounter/gen"
)

//go:embed ui/*
var embeddedUI embed.FS

const BUCKET string = ""

type MyKVCounter struct{}

func (kv *MyKVCounter) IncrementCounter(bucket uint32, key string, amount int32) (uint32, error) {
	kvcounter.WasiLoggingLoggingLog(kvcounter.WasiLoggingLoggingLevelDebug(), "key", key)
	kvcounter.WasiLoggingLoggingLog(kvcounter.WasiLoggingLoggingLevelDebug(), "amount", string(amount))

	var currentValue uint32 = 0
	currentValueGet := kvcounter.WasiKeyvalueReadwriteGet(bucket, key)

	if !currentValueGet.IsErr() {
		b := kvcounter.WasiKeyvalueTypesIncomingValueConsumeSync(currentValueGet.Unwrap())
		if b.IsErr() {
			return 0, errors.New("failed to consume current value")
		}

		bNum, err := strconv.Atoi(string(b.Unwrap()))
		if err != nil {
			bNum = 0
			kvcounter.WasiLoggingLoggingLog(kvcounter.WasiLoggingLoggingLevelError(), "atoi", err.Error())
		}

		currentValue = uint32(bNum)
	}

	newValue := currentValue + uint32(amount)
	outgoingValue := kvcounter.WasiKeyvalueTypesNewOutgoingValue()
	stream := kvcounter.WasiKeyvalueTypesOutgoingValueWriteBodyAsync(outgoingValue)
	if stream.IsErr() {
		return 0, errors.New("failed to write outgoing kv body")
	}

	bsf := kvcounter.WasiIoStreamsBlockingWriteAndFlush(stream.Unwrap(), []byte(strconv.Itoa(int(newValue))))
	if bsf.IsErr() {
		return 0, errors.New("failed to block write and flush")
	}
	_ = kvcounter.WasiKeyvalueReadwriteSet(bucket, key, outgoingValue)
	// TODO: this is throwing an error even though it isn't erroring
	// if res.IsErr() {
	// 	return 103
	// }

	stat := kvcounter.WasiKeyvalueReadwriteGet(bucket, key)
	if stat.IsErr() {
		return 0, errors.New("failed to read value")
	}

	return newValue, nil
}

func (kv *MyKVCounter) Handle(request kvcounter.WasiHttpIncomingHandlerIncomingRequest, response kvcounter.WasiHttpTypesResponseOutparam) {
	method := kvcounter.WasiHttpTypesIncomingRequestMethod(request)

	kvcounter.WasiLoggingLoggingLog(kvcounter.WasiLoggingLoggingLevelDebug(), "msg", "starting handler")

	pathWithQuery := kvcounter.WasiHttpTypesIncomingRequestPathWithQuery(request)
	if pathWithQuery.IsNone() {
		return
	}

	splitPathQuery := strings.Split(pathWithQuery.Unwrap(), "?")

	path := splitPathQuery[0]
	trimmedPath := strings.Split(strings.TrimPrefix(path, "/"), "/")

	kvcounter.WasiLoggingLoggingLog(kvcounter.WasiLoggingLoggingLevelDebug(), "path", path)
	switch {
	case method == kvcounter.WasiHttpTypesMethodGet() && len(trimmedPath) >= 2 && (trimmedPath[0] == "api" && trimmedPath[1] == "counter"):
		bucket := kvcounter.WasiKeyvalueTypesOpenBucket(BUCKET)
		if bucket.IsErr() {
			return
		}

		var newNum uint32
		var err error
		if len(trimmedPath) == 3 && trimmedPath[2] != "" {
			newNum, err = kv.IncrementCounter(bucket.Unwrap(), trimmedPath[2], 1)
			if err != nil {
				writeHttpResponse(response, 500, []kvcounter.WasiHttpTypesTuple2StringListU8TT{{F0: "Content-Type", F1: []byte("application/json")}}, []byte("{\"error\":\""+err.Error()+"\"}"))
				return
			}
		} else {
			newNum, err = kv.IncrementCounter(bucket.Unwrap(), "default", 1)
			if err != nil {
				writeHttpResponse(response, 500, []kvcounter.WasiHttpTypesTuple2StringListU8TT{{F0: "Content-Type", F1: []byte("application/json")}}, []byte("{\"error\":\""+err.Error()+"\"}"))
				return
			}
			kvcounter.WasiLoggingLoggingLog(kvcounter.WasiLoggingLoggingLevelDebug(), "msg", fmt.Sprintf("increment by 1: %d", newNum))
		}

		resp := struct {
			Counter uint32 `json:"counter"`
		}{
			Counter: newNum,
		}

		bResp, err := json.Marshal(resp)
		if err != nil {
			return
		}

		writeHttpResponse(response, 200, []kvcounter.WasiHttpTypesTuple2StringListU8TT{{F0: "Content-Type", F1: []byte("application/json")}}, bResp)
	default:
		if path == "/" {
			path = "ui/index.html"
		} else {
			path = "ui" + path
		}

		page, err := embeddedUI.ReadFile(path)
		if err != nil {
			writeHttpResponse(response, 404, []kvcounter.WasiHttpTypesTuple2StringListU8TT{{F0: "Content-Type", F1: []byte("application/json")}}, []byte("{\"error\":\""+path+": not found\"}"))
			return
		}

		ext := ""
		extSplit := strings.Split(path, ".")
		if len(extSplit) > 1 {
			ext = extSplit[len(extSplit)-1]
		}

		writeHttpResponse(response, 200, []kvcounter.WasiHttpTypesTuple2StringListU8TT{{F0: "Content-Type", F1: []byte(mime.TypeByExtension(ext))}}, page)
	}

}

func writeHttpResponse(responseOutparam kvcounter.WasiHttpTypesResponseOutparam, statusCode uint16, inHeaders []kvcounter.WasiHttpTypesTuple2StringListU8TT, body []byte) {
	kvcounter.WasiLoggingLoggingLog(kvcounter.WasiLoggingLoggingLevelDebug(), "writeHttpResponse", fmt.Sprintf("writing response: len=%d", len(body)))

	headers := kvcounter.WasiHttpTypesNewFields(inHeaders)

	outgoingResponse := kvcounter.WasiHttpTypesNewOutgoingResponse(statusCode, headers)
	if outgoingResponse.IsErr() {
		return
	}

	outgoingStream := kvcounter.WasiHttpTypesOutgoingResponseWrite(outgoingResponse.Unwrap())
	if outgoingStream.IsErr() {
		return
	}

	pollable := kvcounter.WasiIoStreamsSubscribeToOutputStream(outgoingStream.Val)

	bIndex := 0
	for bIndex != len(body) {
		if kvcounter.WasiPollPollPollOneoff([]uint32{pollable})[0] {
			kvcounter.WasiLoggingLoggingLog(kvcounter.WasiLoggingLoggingLevelDebug(), "writeHttpResponse", fmt.Sprintf("inside loop - bIndex: %d", bIndex))

			cw := kvcounter.WasiIoStreamsCheckWrite(outgoingStream.Val)
			if cw.IsErr() {
				return
			}

			kvcounter.WasiLoggingLoggingLog(kvcounter.WasiLoggingLoggingLevelDebug(), "writeHttpResponse", fmt.Sprintf("inside loop - checkWrite: %d", cw.Val))

			if bIndex+int(cw.Val) > len(body) {
				cw.Val = uint64(len(body) - bIndex)
			}

			kvcounter.WasiLoggingLoggingLog(kvcounter.WasiLoggingLoggingLevelDebug(), "writeHttpResponse", fmt.Sprintf("inside loop - writing: %d-%d", bIndex, bIndex+int(cw.Val)))
			w := kvcounter.WasiIoStreamsWrite(outgoingStream.Val, body[bIndex:int(cw.Val)+bIndex])
			if w.IsErr() {
				kvcounter.WasiLoggingLoggingLog(kvcounter.WasiLoggingLoggingLevelError(), "writeHttpResponse", fmt.Sprintf("failed to write to stream: %v", w.UnwrapErr()))
				return
			}

			bIndex += int(cw.Val)
		}
	}

	f := kvcounter.WasiIoStreamsFlush(outgoingStream.Val)
	if f.IsErr() {
		kvcounter.WasiLoggingLoggingLog(kvcounter.WasiLoggingLoggingLevelError(), "writeHttpResponse", fmt.Sprintf("failed to flush to stream: %v", f.UnwrapErr()))
		return
	}

	kvcounter.WasiHttpTypesFinishOutgoingStream(outgoingStream.Val)

	// NOTE: I dont know why we have to do these two steps
	kvcounter.WasiPollPollPollOneoff([]uint32{pollable})
	kvcounter.WasiIoStreamsCheckWrite(outgoingStream.Val)

	outparm := kvcounter.WasiHttpTypesSetResponseOutparam(responseOutparam, outgoingResponse)
	if outparm.IsErr() {
		return
	}
}

func init() {
	mkv := new(MyKVCounter)
	kvcounter.SetExportsWasiHttpIncomingHandler(mkv)
}

//go:generate wit-bindgen tiny-go ./wit -w kvcounter --out-dir=gen
func main() {}
