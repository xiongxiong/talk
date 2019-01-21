package talk_test

import (
	. "github.com/onsi/ginkgo"

	"talk/req"

	"github.com/google/uuid"
)

func init() {
	Describe("connect should work", func() {
		It("should work for single request", func() {
			talk.Serve()
			req := talk.ConnectRequest{
				Req:  req.NewReq(nil),
				UUID: uuid.New(),
			}
			talk.Request(req)
			_, err := req.Res()
			if err != nil {
				t.Errorf("connect failure -- %s", err.Error())
			}
			t.Log("connect success")
		})
		It("should work for mutiple request", func() {

		})
	})
}
