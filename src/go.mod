module code.cloudfoundry.org/system-metrics

go 1.12

require (
	code.cloudfoundry.org/go-batching v0.0.0-20171020220229-924d2a9b48ac
	code.cloudfoundry.org/go-envstruct v1.5.0
	code.cloudfoundry.org/go-loggregator v0.0.0-20190813173818-049b6bf8152a // pinned
	code.cloudfoundry.org/tlsconfig v0.0.0-20191126220907-6c65973656e3
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/golang/protobuf v1.3.2
	github.com/onsi/ginkgo v1.10.3
	github.com/onsi/gomega v1.7.1
	github.com/prometheus/client_golang v1.2.1
	github.com/prometheus/client_model v0.0.0-20191202183732-d1d2010b5bee // indirect
	github.com/prometheus/procfs v0.0.8 // indirect
	github.com/shirou/gopsutil v2.19.11+incompatible
	github.com/square/certstrap v1.2.0 // indirect
	github.com/stretchr/testify v1.4.0 // indirect
	golang.org/x/lint v0.0.0-20190313153728-d0100b6bd8b3 // indirect
	golang.org/x/net v0.0.0-20191209160850-c0dbc17a3553
	golang.org/x/sys v0.0.0-20191218084908-4a24b4065292 // indirect
	golang.org/x/text v0.3.2 // indirect
	golang.org/x/tools v0.0.0-20190524140312-2c0ae7006135 // indirect
	google.golang.org/genproto v0.0.0-20191216205247-b31c10ee225f // indirect
	google.golang.org/grpc v1.26.0
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/yaml.v2 v2.2.7 // indirect
)

replace golang.org/x/sys => golang.org/x/sys v0.0.0-20190801041406-cbf593c0f2f3
