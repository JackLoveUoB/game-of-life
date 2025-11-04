make sure python is installed

Run this command

GO111MODULE=on go get golang.org/x/perf/cmd/benchstat

to download bench stat


run this command:

go test -run=^$ -bench=BenchmarkGol -timeout=100s -cpuprofile=cpu.prof 2>&1 | grep '^Benchmark' | tee results.out

to get the results of the benchmark into a .out file

Then use this to convert to csv:

go run golang.org/x/perf/cmd/benchstat -format csv results.out | tee results.csv

then after that you can run python plot.py to plot the graph, then save the photo.