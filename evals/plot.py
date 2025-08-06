# /// script
# dependencies = [
#   "polars>=0.20.0",
#   "altair>=5.4.0",
#   "vl-convert-python>=1.8.0"
# ]
# ///
# To generate plot: uv run plot.py

import polars as pl
import altair as alt
from pathlib import Path

def main():
    # Read the CSV file
    csv_path = Path("results.csv")
    if not csv_path.exists():
        print(f"Error: {csv_path} not found")
        return

    df = pl.read_csv(csv_path)

    # Get the latest entry for each model (based on date)
    df = df.with_columns(
        pl.col('date').str.to_datetime(),
        pl.col('duration_seconds').cast(pl.Float64)
    )

    # Get the latest entry for each model
    latest_df = df.group_by('model').agg(
        pl.all().sort_by('date').last()
    )

    # Convert accuracy to percentage
    latest_df = latest_df.with_columns(
        (pl.col('total_accuracy') * 100).alias('total_accuracy_pct')
    )

    # Create the scatter plot using Polars' built-in plotting
    scatter = latest_df.plot.scatter(
        x='duration_seconds',
        y='total_accuracy_pct',
        color='model'
    ).encode(
        x=alt.X('duration_seconds:Q', scale=alt.Scale(domain=[0, 225]))  # Extend x-axis to give room for labels
    )

    # Create text labels at 45 degree angle
    labels = latest_df.plot.text(
        x='duration_seconds',
        y='total_accuracy_pct',
        text='model'
    ).mark_text(
        angle=45,
        align="left",
        dx=10,
        fontSize=14
    )

    # Combine scatter and labels
    chart = (scatter + labels).properties(
        title='Total Accuracy vs Duration (Latest Results by Model)',
        width=1000,  # Increased width to accommodate labels
        height=500
    ).resolve_scale(
        color='independent'
    )

    # Save the plot
    chart.save("accuracy_duration.png")
    chart.save("accuracy_duration.html")

    # Print summary data
    print("\nLatest results by model:")
    summary = latest_df.select(['model', 'duration_seconds', 'total_accuracy_pct']).sort('duration_seconds')

    for row in summary.iter_rows(named=True):
        print(f"  {row['model']:<30} {row['duration_seconds']:>8.2f}s  {row['total_accuracy_pct']:>6.1f}%")

if __name__ == "__main__":
    main()
