package talk_test

import (
	"context"
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
			flt := func(keys map[interface{}]interface{}) bool {
				return true
			}
			talk.Connect(context.TODO(), flt)

			Expect(talk.ClientCount()).To(Equal(1))
		})
		It("should work for timeout", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			time.Sleep(3 * time.Second)

			var cli talk.Client
			flt := func(keys map[interface{}]interface{}) bool {
				return true
			}
			select {
			case <-ctx.Done():
			default:
				cli = talk.Connect(ctx, flt)
			}
			Expect(cli).To(BeNil())
		})
		It("should work for mutiple request", func() {
			clis := []talk.Client{}
			connect := func(wg *sync.WaitGroup) {
				defer wg.Done()
				flt := func(keys map[interface{}]interface{}) bool {
					return true
				}
				cli := talk.Connect(context.TODO(), flt)
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
			mutex := sync.RWMutex{}
			return func(keys map[interface{}]interface{}) bool {
				ok := false
				mutex.RLock()
				for k := range keys {
					if k == keyword {
						ok = true
						break
					}
				}
				mutex.RUnlock()
				return ok
			}
		}
		connect := func(key interface{}) talk.Client {
			flt := filter(key)
			cli := talk.Connect(context.TODO(), flt)
			Expect(cli).NotTo(BeNil())
			return cli
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
				keys := make(map[interface{}]interface{})
				keys["hello"] = nil
				for i := 0; i < times; i++ {
					res := talk.Send(context.TODO(), keys, "hello world")
					Expect(res).To(Equal("success"))
				}
			})
		})
		Describe("receive should work", func() {
			It("filter should work", func() {
				go func() {
					ticker := time.NewTicker(time.Second)
					keys := make(map[interface{}]interface{})
					keys["A"] = nil
					for {
						select {
						case <-ticker.C:
							res := talk.Send(context.TODO(), keys, "AA")
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
	Measure("how many connections", func(b Benchmarker) {

	}, 10)
}

func TestTalk(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Talk Suite")
}

func BenchmarkTalk(b *testing.B) {
	talk.Start()
	defer talk.Stop()

	flt := func(map[interface{}]interface{}) bool {
		return true
	}
	for i := 0; i < b.N; i++ {
		talk.Connect(context.TODO(), flt)
	}
}
