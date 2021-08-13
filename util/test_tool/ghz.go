package test_tool

import (
	"fmt"
	"gitee.com/kelvins-io/kelvins/util/middleware"
	"github.com/bojand/ghz/printer"
	"github.com/bojand/ghz/runner"
	"google.golang.org/grpc"
	"io"
	"os"
)

type ReportFormat string

const (
	ReportHTML          ReportFormat = "html"
	ReportCSV           ReportFormat = "csv"
	ReportSummary       ReportFormat = "summary"
	ReportJSON          ReportFormat = "json"
	ReportPretty        ReportFormat = "pretty"
	ReportInfluxSummary ReportFormat = "influx-summary"
	ReportInfluxDetails ReportFormat = "influx-details"
)

type GhzTestOption struct {
	Call         string
	Host         string
	Token        string
	ReportFormat ReportFormat
	Out          io.Writer
	Options      []runner.Option
}

func ExecuteRPCGhzTest(opt *GhzTestOption) error {
	if opt == nil {
		return fmt.Errorf("ghz test opt nil")
	}
	if opt.ReportFormat == "" {
		opt.ReportFormat = ReportHTML
	}
	if opt.Out == nil {
		opt.Out = os.Stdout
	}
	var optsCall []grpc.CallOption
	var optsGhz []runner.Option
	if opt.Token != "" {
		optsCall = append(optsCall, grpc.PerRPCCredentials(middleware.RPCPerCredentials(opt.Token)))
	}
	optsGhz = append(optsGhz, runner.WithDefaultCallOptions(optsCall))
	optsGhz = append(optsGhz, opt.Options...)
	report, err := runner.Run(
		opt.Call,
		opt.Host,
		optsGhz...,
	)
	if err != nil {
		return err
	}
	reportPrinter := printer.ReportPrinter{
		Out:    opt.Out,
		Report: report,
	}
	return reportPrinter.Print(string(opt.ReportFormat))
}
