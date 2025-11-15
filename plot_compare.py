import re
import matplotlib.pyplot as plt
import numpy as np

MOD_FILE = "modulo_results.txt"
NOMOD_FILE = "nomod_results.txt"

def parse_bench_file(path):
    """Extract ns/op values from a go benchmark output file."""
    times_sec = []
    pattern = re.compile(r'BenchmarkGol\S*\s+\d+\s+(\d+)\s+ns/op')

    with open(path, "r") as f:
        for line in f:
            m = pattern.search(line)
            if m:
                ns = int(m.group(1))
                times_sec.append(ns / 1e9)  # convert to seconds
    return times_sec

# Load data
mod_times = parse_bench_file(MOD_FILE)
nomod_times = parse_bench_file(NOMOD_FILE)

# Ensure both lists are same length
n = min(len(mod_times), len(nomod_times))
mod_times = mod_times[:n]
nomod_times = nomod_times[:n]

trials = np.arange(1, n + 1)

# -----------------------
#  LARGE FONT SETTINGS
# -----------------------
plt.rcParams.update({
    "font.size": 30,        # Base font size
    #"axes.titlesize": 24,   # Title size
    "axes.labelsize": 30,   # Axis label size
    "xtick.labelsize": 30,  # Tick label size
    "ytick.labelsize": 30,
    "legend.fontsize": 30,
})

fig, ax = plt.subplots(figsize=(10, 10))  # Bigger figure for reports

width = 0.35
x = np.arange(n)

# Bars
bars_mod = ax.bar(x - width/2, mod_times, width, label="Modulo")
bars_nomod = ax.bar(x + width/2, nomod_times, width, label="Non-Modulo")

# Labels & Title
ax.set_xlabel("Trial Number")
ax.set_ylabel("Benchmark Time (seconds)")
#ax.set_title("Modulo vs Non-Modulo Performance (512×512×1000, 5 Runs)")
ax.set_xticks(x)
ax.set_xticklabels(trials)
ax.set_ylim(bottom=0)
ax.legend()

ax.grid(axis="y", linestyle="--", alpha=0.4)

plt.tight_layout()
plt.savefig("mod_vs_nomod_bar_large.png", dpi=300)
plt.show()
