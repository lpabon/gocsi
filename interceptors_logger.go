package gocsi

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// LoggingOption configures the logging interceptor.
type LoggingOption func(*loggingOpts)

type loggingOpts struct {
	reqw io.Writer
	repw io.Writer
}

// WithRequestLogging is a LoggingOption that enables request logging
// for the logging interceptor.
func WithRequestLogging(w io.Writer) LoggingOption {
	return func(o *loggingOpts) {
		if w == nil {
			w = os.Stdout
		}
		o.reqw = w
	}
}

// WithResponseLogging is a LoggingOption that enables response logging
// for the logging interceptor.
func WithResponseLogging(w io.Writer) LoggingOption {
	return func(o *loggingOpts) {
		if w == nil {
			w = os.Stdout
		}
		o.repw = w
	}
}

type loggingInterceptor struct {
	opts loggingOpts
}

// NewServerLogger returns a new UnaryServerInterceptor that can be
// configured to log both request and response data.
func NewServerLogger(
	opts ...LoggingOption) grpc.UnaryServerInterceptor {

	return newLoggingInterceptor(opts...).handleServer
}

// NewClientLogger provides a UnaryClientInterceptor that can be
// configured to log both request and response data.
func NewClientLogger(
	opts ...LoggingOption) grpc.UnaryClientInterceptor {

	return newLoggingInterceptor(opts...).handleClient
}

func newLoggingInterceptor(opts ...LoggingOption) *loggingInterceptor {
	i := &loggingInterceptor{}
	for _, withOpts := range opts {
		withOpts(&i.opts)
	}
	return i
}

func (s *loggingInterceptor) handleServer(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (interface{}, error) {

	return s.handle(ctx, info.FullMethod, req, func() (interface{}, error) {
		return handler(ctx, req)
	})
}

func (s *loggingInterceptor) handleClient(
	ctx context.Context,
	method string,
	req, rep interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption) error {

	_, err := s.handle(ctx, method, req, func() (interface{}, error) {
		return rep, invoker(ctx, method, req, rep, cc, opts...)
	})
	return err
}

func (s *loggingInterceptor) handle(
	ctx context.Context,
	method string,
	req interface{},
	next func() (interface{}, error)) (rep interface{}, failed error) {

	// If the request is nil then pass control to the next handler
	// in the chain.
	if req == nil {
		return next()
	}

	w := &bytes.Buffer{}
	reqID, reqIDOK := GetRequestID(ctx)

	// Print the request
	fmt.Fprintf(w, "%s: ", method)
	if reqIDOK {
		fmt.Fprintf(w, "REQ %04d", reqID)
	}
	rprintReqOrRep(w, req)
	fmt.Fprintln(s.opts.reqw, w.String())

	w.Reset()

	// Get the response.
	rep, failed = next()

	if s.opts.repw == nil {
		return
	}

	// Print the response method name.
	fmt.Fprintf(w, "%s: ", method)
	if reqIDOK {
		fmt.Fprintf(w, "REP %04d", reqID)
	}

	// Print the response error if it is set.
	if failed != nil {
		fmt.Fprint(w, ": ")
		fmt.Fprint(w, failed)
	}

	// Print the response data if it is set.
	if rep != nil {
		rprintReqOrRep(w, rep)
	}
	fmt.Fprintln(s.opts.repw, w.String())

	return
}

var emptyValRX = regexp.MustCompile(
	`^((?:)|(?:\[\])|(?:<nil>)|(?:map\[\]))$`)

// rprintReqOrRep is used by the server-side interceptors that log
// requests and responses.
func rprintReqOrRep(w io.Writer, obj interface{}) {
	rv := reflect.ValueOf(obj).Elem()
	tv := rv.Type()
	nf := tv.NumField()
	printedColon := false
	printComma := false
	for i := 0; i < nf; i++ {
		name := tv.Field(i).Name
		if name == "UserCredentials" {
			continue
		}
		sv := fmt.Sprintf("%v", rv.Field(i).Interface())
		if emptyValRX.MatchString(sv) {
			continue
		}
		if printComma {
			fmt.Fprintf(w, ", ")
		}
		if !printedColon {
			fmt.Fprintf(w, ": ")
			printedColon = true
		}
		printComma = true
		fmt.Fprintf(w, "%s=%s", name, sv)
	}
}
