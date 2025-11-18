import sys
import matplotlib.pyplot as plt

plt.rcParams.update({
    "font.size": 20,
    "axes.labelsize": 20,
    "xtick.labelsize": 20,
    "ytick.labelsize": 15,
    "legend.fontsize": 25
})

def parse_file(path: str):
    """
    Parse a whitespace-separated benchmark file with lines like:

    BenchmarkGol/512x512x1000-1-16          2   10152768548 ns/op
    """

    threads_to_times = {}

    with open(path, "r") as f:
        for line in f:
            line = line.strip()
            if not line.startswith("Benchmark"):
                continue

            parts = line.split()
            if len(parts) < 4:
                continue

            name = parts[0]

            try:
                time_ns = int(parts[-2])
            except ValueError:
                continue

            bits = name.split("-")
            if len(bits) < 2:
                continue

            try:
                threads = int(bits[-2])
            except ValueError:
                continue

            threads_to_times.setdefault(threads, []).append(time_ns)

    threads_sorted = sorted(threads_to_times.keys())
    times_seconds = [
        (sum(threads_to_times[t]) / len(threads_to_times[t])) / 1e9
        for t in threads_sorted
    ]

    return threads_sorted, times_seconds


def main():
    if len(sys.argv) < 6:
        print("Usage: python plot_compare_line.py file1 file2 file3 [label1 label2 label3]")
        sys.exit(1)

    file1 = sys.argv[1]
    file2 = sys.argv[2]
    file3 = sys.argv[3]
    file4 = sys.argv[4]
    file5 = sys.argv[5]

    label1 = sys.argv[6] if len(sys.argv) > 4 else "Run 1"
    label2 = sys.argv[7] if len(sys.argv) > 5 else "Run 2"
    label3 = sys.argv[8] if len(sys.argv) > 6 else "Run 3"
    label4 = sys.argv[9] if len(sys.argv) > 7 else "Run 4"
    label5 = sys.argv[10] if len(sys.argv) > 8 else "Run 5"

    threads1, times1 = parse_file(file1)
    threads2, times2 = parse_file(file2)
    threads3, times3 = parse_file(file3)
    threads4, times4 = parse_file(file4)
    threads5, times5 = parse_file(file5)

    plt.figure(figsize=(12, 10))
    plt.plot(threads1, times1, marker="o", label=label1)
    plt.plot(threads2, times2, marker="o", label=label2)
    plt.plot(threads3, times3, marker="o", label=label3)
    plt.plot(threads4, times4, marker="o", label=label4)
    plt.plot(threads5, times5, marker="o", label=label5)

    plt.xticks(range(1, 17, 1))
    plt.yticks(range(0, 12, 1))

    plt.xlabel("Worker threads")
    plt.ylabel("Time per run (seconds)")
    plt.title("")
    plt.grid(True, linestyle="--", alpha=0.5)
    plt.legend()
    plt.tight_layout()
    plt.show()


if __name__ == "__main__":
    main()
