package tools

import (
	"go-fly-muti/frpc"
	"testing"
)

func TestRPCServiceSendToVisitor(t *testing.T) {
	service := new(frpc.Service)
	reply := new(frpc.Reply)
	err := service.SendToVisitor(frpc.Message{
		VisitorId: "visitor-test",
		Content:   "hello",
	}, reply)
	if err != nil {
		t.Fatalf("SendToVisitor() error = %v", err)
	}
	if reply.Code != "200" || reply.Msg != "ok" {
		t.Fatalf("SendToVisitor() reply = %+v", reply)
	}
}
