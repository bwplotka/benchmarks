// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package localgrpc

import (
	"context"
	"fmt"
	"io"
	"iter"
	"log"
	"net"
	"os"
	"testing"
	"time"

	listv0 "github.com/bwplotka/benchmarks/benchmarks/local-grpc/dev/bwplotka/list/v0"
	"github.com/efficientgo/core/errors"
	"github.com/efficientgo/core/testutil"
	"github.com/fullstorydev/grpchan/inprocgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var _ listv0.ListStringsServer = &List{}

type List struct {
	listv0.UnimplementedListStringsServer

	resps []*listv0.ListResponse
}

func (l *List) List(_ *listv0.ListRequest, srv grpc.ServerStreamingServer[listv0.ListResponse]) error {
	for _, r := range l.resps {
		if err := srv.Send(r); err != nil {
			return err
		}
	}

	return nil
}

/*
	export bench=bench01-2025 && go test \
	  -run '^$' -bench '^BenchmarkLocal' \
	  -benchtime 5s -count 6 -cpu 2 -timeout 999m \
	  | tee ${bench}.txt

benchstat -col /impl bench12-2024.txt
*/
func BenchmarkLocal(b *testing.B) {
	benchmarkLocal(testutil.NewTB(b))
}

func TestBenchmarkLocal(t *testing.T) {
	benchmarkLocal(testutil.NewTB(t))
}

const testString = "1pjpwqp23-wqwpqj--jwfewpjfpwjfq0jJWQIFnI230NInp@#(lfpsakon[]la,','-K-21J1J3RJFmx/,mx?wpjfjpw1-)DSPL!"

func benchmarkLocal(b testutil.TB) {
	// Always send ~10MB worth data, so 10k x 100B strings.
	list := make([]string, 1e4)
	for i := range list {
		list[i] = testString
	}
	for _, respSize := range []int{1, 10, 100} {
		for _, implCase := range []struct {
			name     string
			clientFn func(b testutil.TB, srv listv0.ListStringsServer) listv0.ListStringsClient
		}{
			{name: "localhost", clientFn: newLocalhostClient},
			{name: "unixsocket", clientFn: newUnixSocketClient},
			{name: "grpchannel", clientFn: newGRPCChannelClient},
			{name: "grpchannel-nocpy", clientFn: newGRPCChannelClientNoCpy},
			{name: "chan", clientFn: newGoChanClient},
			{name: "buffer", clientFn: newBufferedClient},
			{name: "iter", clientFn: newIterClient},
		} {
			b.Run(fmt.Sprintf("respSize=%v/impl=%v", respSize, implCase.name), func(b testutil.TB) {
				testSrv := &List{resps: make([]*listv0.ListResponse, 0, len(list)/respSize)}
				for i := 0; i < len(list); i += respSize {
					testSrv.resps = append(testSrv.resps, &listv0.ListResponse{Strings: list[i : i+respSize]})
				}
				client := implCase.clientFn(b, testSrv)

				var (
					ctx = context.Background()
					req = &listv0.ListRequest{}
					res = &listv0.ListResponse{}

					gotStrings []string
				)

				b.ReportAllocs()
				b.ResetTimer()
				for range b.N() {
					stream, err := client.List(ctx, req)
					testutil.Ok(b, err)

					for {
						err := stream.RecvMsg(res)
						if err == io.EOF {
							break
						}
						testutil.Ok(b, err)

						if !b.IsBenchmark() {
							testutil.Assert(b, respSize == len(res.Strings))
							gotStrings = append(gotStrings, res.GetStrings()...)
						}
					}
				}

				if !b.IsBenchmark() {
					testutil.Equals(b, list, gotStrings)
				}
			})
		}
	}
}

func newLocalhostClient(b testutil.TB, srv listv0.ListStringsServer) listv0.ListStringsClient {
	l, err := net.Listen("tcp", "localhost:0")
	testutil.Ok(b, err)

	insec := insecure.NewCredentials()
	grpcSrv := grpc.NewServer(grpc.Creds(insec))
	listv0.RegisterListStringsServer(grpcSrv, srv)

	go func() { _ = grpcSrv.Serve(l) }()
	b.Cleanup(grpcSrv.Stop)

	// Purposefully using grpc.WithBlock despite deprecation, for the best effort "warm up".
	cc, err := grpc.NewClient(l.Addr().String(), grpc.WithTransportCredentials(insec), grpc.WithBlock())
	testutil.Ok(b, err)
	return listv0.NewListStringsClient(cc)
}

func newUnixSocketClient(b testutil.TB, srv listv0.ListStringsServer) listv0.ListStringsClient {
	const sockAddr = "/tmp/grpc.sock"

	if _, err := os.Stat(sockAddr); !os.IsNotExist(err) {
		if err := os.RemoveAll(sockAddr); err != nil {
			log.Fatal(err)
		}
	}
	l, err := net.Listen("unix", sockAddr)
	testutil.Ok(b, err)

	insec := insecure.NewCredentials()
	grpcSrv := grpc.NewServer(grpc.Creds(insec))
	listv0.RegisterListStringsServer(grpcSrv, srv)

	go func() { _ = grpcSrv.Serve(l) }()
	b.Cleanup(grpcSrv.Stop)

	// Purposefully using grpc.WithBlock despite deprecation, for the best effort "warm up".
	cc, err := grpc.NewClient(
		"unix:"+sockAddr,
		grpc.WithTransportCredentials(insec),
		grpc.WithBlock(),
	)
	testutil.Ok(b, err)
	return listv0.NewListStringsClient(cc)
}

func newGRPCChannelClient(_ testutil.TB, srv listv0.ListStringsServer) listv0.ListStringsClient {
	ch := &inprocgrpc.Channel{}

	listv0.RegisterListStringsServer(ch, srv)
	return listv0.NewListStringsClient(ch)
}

type nopCloner struct{}

func (nopCloner) Copy(out, in interface{}) error {
	out = in
	return nil
}

func (nopCloner) Clone(in interface{}) (out interface{}, _ error) {
	return in, nil
}

func newGRPCChannelClientNoCpy(_ testutil.TB, srv listv0.ListStringsServer) listv0.ListStringsClient {
	ch := &inprocgrpc.Channel{}
	ch = ch.WithCloner(nopCloner{})

	listv0.RegisterListStringsServer(ch, srv)
	return listv0.NewListStringsClient(ch)
}

func newBufferedClient(_ testutil.TB, srv listv0.ListStringsServer) listv0.ListStringsClient {
	return &bufferedClient{srv: srv}
}

type bufferedClient struct {
	srv listv0.ListStringsServer
}

func (b *bufferedClient) List(ctx context.Context, in *listv0.ListRequest, _ ...grpc.CallOption) (listv0.ListStrings_ListClient, error) {
	buf := &buffer{ctx: ctx} // buf: make([]*listv0.ListResponse, 0, 1e4)} This makes perResponse=1 much better, but =100 much worse (:
	return buf, b.srv.List(in, buf)
}

type buffer struct {
	listv0.ListStrings_ListServer
	listv0.ListStrings_ListClient

	ctx context.Context

	buf      []*listv0.ListResponse
	received int
}

func (b *buffer) Context() context.Context {
	return b.ctx
}

func (b *buffer) SendMsg(m any) error {
	return b.Send(m.(*listv0.ListResponse))
}

func (b *buffer) Send(resp *listv0.ListResponse) error {
	b.buf = append(b.buf, resp)
	return nil
}

func (b *buffer) RecvMsg(m any) error {
	r, err := b.Recv()
	if err != nil {
		return err
	}

	m.(*listv0.ListResponse).Strings = r.Strings
	return nil
}

func (b *buffer) Recv() (*listv0.ListResponse, error) {
	if b.received >= len(b.buf) {
		return nil, io.EOF

	}
	r := b.buf[b.received]
	b.received++
	return r, nil
}

func newGoChanClient(_ testutil.TB, srv listv0.ListStringsServer) listv0.ListStringsClient {
	return &chanClient{srv: srv}
}

type chanClient struct {
	srv listv0.ListStringsServer
}

func (c *chanClient) List(ctx context.Context, in *listv0.ListRequest, _ ...grpc.CallOption) (listv0.ListStrings_ListClient, error) {
	buf := &channel{ctx: ctx, ch: make(chan *listv0.ListResponse), errCh: make(chan error)}

	go func() {
		if err := c.srv.List(in, buf); err != nil {
			buf.errCh <- err
		}
		close(buf.ch)
		close(buf.errCh)
	}()
	return buf, nil
}

type channel struct {
	listv0.ListStrings_ListServer
	listv0.ListStrings_ListClient

	ctx context.Context

	ch    chan *listv0.ListResponse
	errCh chan error
}

func (c *channel) Context() context.Context {
	return c.ctx
}

func (c *channel) SendMsg(m any) error {
	return c.Send(m.(*listv0.ListResponse))
}

func (c *channel) Send(resp *listv0.ListResponse) error {
	c.ch <- resp
	return nil
}

func (c *channel) RecvMsg(m any) error {
	r, err := c.Recv()
	if err != nil {
		return err
	}

	m.(*listv0.ListResponse).Strings = r.Strings
	return nil
}

func (c *channel) Recv() (*listv0.ListResponse, error) {
	select {
	case r, ok := <-c.ch:
		if !ok {
			return nil, io.EOF
		}
		return r, nil
	case err, ok := <-c.errCh:
		if !ok {
			return nil, io.EOF
		}
		return nil, err
	}
}

func newIterClient(_ testutil.TB, srv listv0.ListStringsServer) listv0.ListStringsClient {
	return &iterClient{srv: srv}
}

type iterClient struct {
	srv listv0.ListStringsServer
}

func (c *iterClient) List(ctx context.Context, in *listv0.ListRequest, _ ...grpc.CallOption) (listv0.ListStrings_ListClient, error) {
	y := &yielder{ctx: ctx}

	// Pull from iter.Seq2[*listv0.ListResponse, error].
	y.recv, y.stop = iter.Pull2(func(yield func(*listv0.ListResponse, error) bool) {
		y.send = yield
		if err := c.srv.List(in, y); err != nil {
			yield(nil, err)
			return
		}
	})
	return y, nil
}

type yielder struct {
	listv0.ListStrings_ListServer
	listv0.ListStrings_ListClient

	ctx context.Context

	send func(*listv0.ListResponse, error) bool
	recv func() (*listv0.ListResponse, error, bool)
	stop func()
}

func (y *yielder) Context() context.Context {
	return y.ctx
}

func (y *yielder) SendMsg(m any) error {
	return y.Send(m.(*listv0.ListResponse))
}

func (y *yielder) RecvMsg(m any) error {
	r, err := y.Recv()
	if err != nil {
		return err
	}

	m.(*listv0.ListResponse).Strings = r.Strings
	return nil
}

func (y *yielder) Send(resp *listv0.ListResponse) error {
	if !y.send(resp, nil) {
		return errors.New("iterator stopped receiving")
	}
	return nil
}

func (y *yielder) Recv() (*listv0.ListResponse, error) {
	r, err, ok := y.recv()
	if err != nil {
		y.stop()
		return nil, err
	}
	if !ok {
		return nil, io.EOF
	}
	return r, nil
}

type server func(send func(any))

func serve(send func(any)) {
	send("yo")
	time.Sleep(1 * time.Second) // Imagine a processing time.
	send("are you there?")
}

func call(serve server) {
	serve(func(s any) {
		fmt.Println(time.Now().Second(), s)
	})
}

func TestCall(t *testing.T) {
	call(serve)
}

type pullClient func(recv func() (any, bool))

var _ pullClient = pullCall

func pullCall(recv func() (any, bool)) {
	for {
		m, ok := recv()
		if !ok {
			return
		}
		fmt.Println(time.Now().Second(), m)
	}
}

func TestPullCall_Buffer(t *testing.T) {
	// We have to allow pushing somewhere...
	var medium []any
	serve(func(m any) {
		medium = append(medium, m)
	})

	var i int
	pullCall(func() (m any, ok bool) {
		if i < len(medium) {
			i++
			return medium[i-1], true
		}
		return nil, false
	})
}

func TestPullCall_Iter(t *testing.T) {
	var iterator iter.Seq[any] = func(yield func(any) bool) {
		serve(func(a any) {
			yield(a)
		})
	}

	recv, _ := iter.Pull(iterator)
	pullCall(recv)
}
