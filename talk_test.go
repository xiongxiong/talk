package talk_test

import (
	"sync"
	"talk"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func init() {
	Describe("connect should work", func() {
		BeforeEach(func() {
			talk.Start()
		})
		AfterEach(func() {
			talk.Stop()
		})
		It("should work for single request", func() {
			filter := func(keys []interface{}) bool {
				return true
			}
			talk.Connect([]talk.Filter{filter})

			Expect(talk.ClientCount()).To(Equal(1))
		})
		It("should work for mutiple request", func() {
			clis := []talk.Client{}
			connect := func(wg *sync.WaitGroup) {
				defer wg.Done()
				filter := func(keys []interface{}) bool {
					return true
				}
				cli, err := talk.Connect([]talk.Filter{filter})
				Expect(err).To(BeNil())
				Expect(cli).NotTo(BeNil())
				clis = append(clis, cli)
			}

			wg := sync.WaitGroup{}
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go connect(&wg)
			}
			wg.Wait()
			Expect(talk.ClientCount()).To(Equal(10))

			for i := 0; i < 3; i++ {
				clis[i].Close()
			}
			time.Sleep(time.Second)
			Expect(talk.ClientCount()).To(Equal(7))
		})
	})
	Describe("message should work", func() {
		filter := func(keyword interface{}) talk.Filter {
			return func(keys []interface{}) bool {
				ok := false
				for _, key := range keys {
					if key == keyword {
						ok = true
						break
					}
				}
				return ok
			}
		}
		connect := func(key interface{}) talk.Conn {
			flt := filter(key)
			cli, err := talk.Connect([]talk.Filter{flt})
			Expect(err).To(BeNil())
			Expect(cli).NotTo(BeNil())
			return cli.GetConn(&flt)
		}

		BeforeEach(func() {
			talk.Start()
		})
		AfterEach(func() {
			talk.Stop()
		})
		Describe("send should work", func() {
			It("should work when there is no client", func() {
				times := 10
				for i := 0; i < times; i++ {
					res := talk.Send([]interface{}{"hello"}, "hello world")
					Expect(res).To(Equal("success"))
				}
			})
		})
		Describe("receive should work", func() {
			It("filter should work", func() {
				go func() {
					ticker := time.NewTicker(time.Second)
					for {
						select {
						case <-ticker.C:
							res := talk.Send([]interface{}{"A"}, "AA")
							Expect(res).To(Equal("success"))
						}
					}
				}()
				connA := connect("A")
				Expect(connA).ToNot(BeNil())
				connB := connect("B")
				Expect(connB).ToNot(BeNil())

				done := make(chan struct{})
				var times int64 = 5
				go func() {
					defer GinkgoRecover()

					var msgStamp int64 = 0
					for {
						select {
						case <-done:
							break
						case msg := <-connA.C():
							Expect(msg.Content).To(Equal("AA"))
							Expect(msg.MsgStamp).To(BeNumerically(">", msgStamp))
							msgStamp = msg.MsgStamp
							if msgStamp == times {
								break
							}
						}
					}
					Expect(msgStamp).To(Equal(times))
				}()
				go func() {
					defer GinkgoRecover()

					var msgStamp int64 = 0
					for {
						select {
						case <-done:
							break
						case msg := <-connB.C():
							Expect(msg.Content).ToNot(Equal("AA"))
							Expect(msg.MsgStamp).To(BeNumerically(">", msgStamp))
							msgStamp = msg.MsgStamp
						}
					}
					Expect(msgStamp).To(Equal(0))
				}()

				time.Sleep(10 * time.Second)
				done <- struct{}{}
			})
		})
	})
}

func TestConnect(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Connect Suite")
}
