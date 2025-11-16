import pandas as pd
import matplotlib.pyplot as plt

# Read whitespace-separated benchmark output:
#   BenchmarkGol/512x512x1000-1-16   1   18924333933 ns/op
benchmark_data = pd.read_csv(
    "results.csv",
    delim_whitespace=True,
    header=None,
    names=["name", "iters", "time_ns", "unit"],
)

# Keep only our Gol benchmark lines (paranoid filter)
benchmark_data = benchmark_data[
    benchmark_data["name"].str.startswith("BenchmarkGol/512x512x1000-")
]

# Convert time_ns to numeric (it’s just the integer number of ns/op)
benchmark_data["time_ns"] = pd.to_numeric(benchmark_data["time_ns"], errors="coerce")

# Drop anything that failed to parse (should be none if the file is clean)
benchmark_data = benchmark_data.dropna(subset=["time_ns"])

# Convert to seconds per op for readability
benchmark_data["time_sec"] = benchmark_data["time_ns"] / 1e9

# Extract number of threads from e.g. BenchmarkGol/512x512x1000-8-16
benchmark_data["threads"] = (
    benchmark_data["name"]
    .str.extract(r"BenchmarkGol/512x512x1000-(\d+)-16")[0]
    .astype(int)
)

# Sort by thread count so the line graph isn’t scrambled
benchmark_data = benchmark_data.sort_values("threads")

print(benchmark_data)

# ---- Line graph: threads vs time_sec ----
plt.figure(figsize=(8, 5))
plt.plot(
    benchmark_data["threads"],
    benchmark_data["time_sec"],
    marker="o",
)

plt.xlabel("Worker threads used")
plt.ylabel("Time per run (seconds per op)")
plt.title("Game of Life 512x512x1000: time vs threads")
plt.grid(True)
plt.xticks(benchmark_data["threads"])  # discrete ticks at 1..16

plt.tight_layout()
plt.savefig("benchmark_threads_vs_time.png", dpi=200)
plt.show()
