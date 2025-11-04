import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns

# Read in the CSV produced from the benchmark.
# - skip the first 2 rows (metadata)
# - no header row in the actual data section, so header=None
# - manually name the columns
benchmark_data = pd.read_csv(
    'results.csv',
    skiprows=2,
    header=None,
    names=['name', 'time', 'ci']
)

# Keep only the rows that are actual benchmark lines (start with 'Gol/')
benchmark_data = benchmark_data[benchmark_data['name'].str.startswith('Gol/')]

# Convert time column (already in seconds per op) to numeric
benchmark_data['time'] = pd.to_numeric(benchmark_data['time'])

# Extract the number of threads from names like 'Gol/512x512x1000-4-16'
# This regex grabs the "4" from that example.
benchmark_data['threads'] = (
    benchmark_data['name']
    .str.extract(r'Gol/512x512x1000-(\d+)-16')[0]
    .astype(int)
)

# (Optional but nice): sort by thread count so the x-axis is ordered
benchmark_data = benchmark_data.sort_values('threads')

print(benchmark_data)

# Plot a bar chart: threads vs time (sec/op)
ax = sns.barplot(
    data=benchmark_data,
    x='threads',
    y='time',
    hue='threads',
    palette='tab10',
    legend=False
)

# Set descriptive axis labels.
ax.set(
    xlabel='Worker threads used',
    ylabel='Time per run (sec/op)'
)

plt.tight_layout()
plt.show()
