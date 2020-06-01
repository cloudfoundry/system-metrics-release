module code.cloudfoundry.org/system-metrics

go 1.12

require (
	code.cloudfoundry.org/go-batching v0.0.0-20171020220229-924d2a9b48ac
	code.cloudfoundry.org/go-envstruct v1.5.0
	code.cloudfoundry.org/go-loggregator v0.0.0-20190813173818-049b6bf8152a // pinned
	code.cloudfoundry.org/tlsconfig v0.0.0-20200131000646-bbe0f8da39b3
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/golang/protobuf v1.4.2 // pinned
	github.com/kr/pretty v0.2.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/onsi/ginkgo v1.12.3
	github.com/onsi/gomega v1.10.1
	github.com/prometheus/client_golang v1.6.0
	github.com/shirou/gopsutil v2.20.5+incompatible
	golang.org/x/net v0.0.0-20200528225125-3c3fba18258b
	golang.org/x/sys v0.0.0-20200523222454-059865788121 // indirect
	google.golang.org/genproto v0.0.0-20200601130524-0f60399e6634 // indirect
	google.golang.org/grpc v1.29.1
)

replace golang.org/x/sys => golang.org/x/sys v0.0.0-20190801041406-cbf593c0f2f3
