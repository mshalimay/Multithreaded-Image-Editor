// Compute average times and speedups for the different modes and data directories
// and plot the speedups for each mode.

package main
import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"sort"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

//=============================================================================
// Data struct and methods to compute average, best times and speedups
//=============================================================================

// Data struct to parse JSON results file
type Data struct {
	Mode		 string  `json:"mode"`
	Threads      int     `json:"threads"`
	TimeElapsed  float64 `json:"timeElapsed"`
	TimeParallel float64 `json:"timeParallel"`
	DataDir      string  `json:"datadir"`
}


// ParseResults parses the 'results.txt' file and returns a map of Data structs
func ParseResults(pathToResultsFile string) map[string][]Data {
	file, _ := os.Open(pathToResultsFile)
	defer file.Close()

	decoder := json.NewDecoder(file)
	dataSets := make(map[string][]Data)

	for {
		var data Data
		if err := decoder.Decode(&data); err != nil {
			fmt.Println(err)
			break
		}
		dataSets[data.Mode] = append(dataSets[data.Mode], data)
	}
	return dataSets
}

// `ComputeAverageTimes` computes the average times for each mode, data directory and number of threads.
// @dataSets: map of Data structs
// returns: map of average times for each mode, data directory and number of threads
// e.g. map["parfiles"]["b"]["4"] = 100 (parfiles took on average 100 seconds to run on data directory "b" with 4 threads)
func ComputeAverageTimes(dataSets map[string][]Data, averagesPath string) map[string]map[string]map[int]float64 {
	averagesElapsed := make(map[string]map[string]map[int]float64, 0)
	counters := make(map[string]map[string]map[int]int, 0)

	// iterate over modes
	for mode, dataSet := range dataSets {
		averagesElapsed[mode] = make(map[string]map[int]float64)
		counters[mode] = make(map[string]map[int]int)
		
		// iterate over datapoints for each mode
		for _, data := range dataSet {

			// initialize dataDir map to populate with averages
			if averagesElapsed[mode][data.DataDir] == nil {
				averagesElapsed[mode][data.DataDir] = make(map[int]float64)
				counters[mode][data.DataDir] = make(map[int]int)
			}
			// accumulate sum and counter for each dataDir and number of threads
			averagesElapsed[mode][data.DataDir][data.Threads] += data.TimeElapsed
			counters[mode][data.DataDir][data.Threads]++
		}

		// compute averages for each dataDir and number of threads for a given mode
		for dataDir, data := range averagesElapsed[mode] {
			for threads, timeElapsed := range data {
				averagesElapsed[mode][dataDir][threads] = timeElapsed / float64(counters[mode][dataDir][threads])
			}
		}		
	}
	// write best times to file
	saveToFile(averagesElapsed, averagesPath)
	return averagesElapsed
}

// `ComputeBestTimes` computes the best times for each mode, data directory and number of threads.
// @dataSets: map of Data structs
// returns: map of best times for each mode, data directory and number of threads
// e.g. map["parfiles"]["b"]["4"] = 100 (parfiles took on 100 seconds to run on data directory "b" with 4 threads on its best run)
func ComputeBestTimes(dataSets map[string][]Data, bestTotalTimesPath, bestParallTimesPath string) map[string]map[string]map[int]float64 {
	bestTotalTimes := make(map[string]map[string]map[int]float64)
	bestParallTimes := make(map[string]map[string]map[int]float64)

	// iterate over modes
	for mode, data := range dataSets {
		// iterate over datapoints for each mode
		for _, data := range data {
			if bestTotalTimes[mode] == nil {
				bestTotalTimes[mode] = make(map[string]map[int]float64)
				bestParallTimes[mode] = make(map[string]map[int]float64)
			}

			if bestTotalTimes[mode][data.DataDir] == nil {
				bestTotalTimes[mode][data.DataDir] = make(map[int]float64)
				bestParallTimes[mode][data.DataDir] = make(map[int]float64)
			}

			// if total time elapsed is less than the current best time for a thread, update best time
			if data.TimeElapsed < bestTotalTimes[mode][data.DataDir][data.Threads] || bestTotalTimes[mode][data.DataDir][data.Threads] == 0 {
				bestTotalTimes[mode][data.DataDir][data.Threads] = data.TimeElapsed
				bestParallTimes[mode][data.DataDir][data.Threads] = data.TimeParallel
			}
		}				
	}

	// Save the results to file
	saveToFile(bestTotalTimes, bestTotalTimesPath)
	saveToFile(bestParallTimes, bestParallTimesPath)

	return bestTotalTimes
}

// `ComputeSpeedups` computes the speedups for each mode, data directory and number of threads.
// @times: map of times for each mode, data directory and number of threads
// returns: map of speedups for each mode, data directory and number of threads
// e.g. map["parfiles"]["b"]["4"] = 2 (parfiles with 4 threads processed data directory "b" on average 2 times faster than sequential impl.)
func ComputeSpeedups(times map[string]map[string]map[int]float64, speedUpsPath string) map[string]map[string]map[int]float64 {
	speedups := make(map[string]map[string]map[int]float64, 0)
	
	// iterate over modes
	for mode, data := range times {
		if mode == "s" {
			continue
		}
		// for each mode create a new map of average speedups per data directory and number of threads
		// e.g. map["parfiles"]["b"]["4"], map["parfiles"]["b"]["8"], etc.
		speedups[mode] = make(map[string]map[int]float64)
		// iterate over data directories
		for dataDir, data := range data {
			// initialize dataDir map to populate with speedups
			speedups[mode][dataDir] = make(map[int]float64)
			for threads, timeElapsed := range data {
				if threads != 1 {
					// speedup = sequential time / parallel time
					speedups[mode][dataDir][threads] = times["s"][dataDir][1] / timeElapsed
				}
			}
		}
	}
	// write speedups to file
	file, _ := os.Create(speedUpsPath)
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.Encode(speedups)
	return speedups
}


//=============================================================================
// Plotting methods
//=============================================================================
// Customized tick marks for the Y axis
type CustomYTicks struct{}

// forces plotter to show all valus in Y axis
func (CustomYTicks) Ticks(min, max float64) []plot.Tick {
	var newTicks []plot.Tick
	defaultTicks := plot.DefaultTicks{}
	ticks := defaultTicks.Ticks(min, max)
	for _, t := range ticks {
		t.Label = fmt.Sprintf("%.2f", t.Value)
		newTicks = append(newTicks, t)
	}
	return newTicks
}

// customized tick marks for the X axis
type CustomXTicks struct{
	Threads []int
}
// forces plotter to show all number in X axis for which there are values
func (t CustomXTicks) Ticks(min, max float64) []plot.Tick {
	var ticks []plot.Tick
	for _, thread := range t.Threads {
		if float64(thread) >= min && float64(thread) <= max {
			ticks = append(ticks, plot.Tick{Value: float64(thread), Label: fmt.Sprintf("%d", thread)})
		}
	}
	return ticks
}


func saveToFile(data map[string]map[string]map[int]float64, path string) {
    file, err := os.Create(path)
    if err != nil {
        panic(err)
    }
    defer file.Close()

    encoder := json.NewEncoder(file)

    for key, val := range data {
        singleRecord := make(map[string]map[string]map[int]float64)
        singleRecord[key] = val

        err = encoder.Encode(singleRecord)
        if err != nil {
            panic(err)
        }

        _, err = file.WriteString("\n")
        if err != nil {
            panic(err)
        }
    }
}



//=============================================================================
// Main
//=============================================================================
func main() {
	// parse command line arguments
	var partial_path, resultsPath string
	
	// os.Args = []string{"", "few"}

	if len(os.Args) >= 2 {
		benchmark_subdir := os.Args[1]
		partial_path = fmt.Sprintf("./benchmark/%s/", benchmark_subdir)
		resultsPath = fmt.Sprintf("./benchmark/results_%s.txt", os.Args[1])

	} else {
		partial_path = "./benchmark/"
		resultsPath = fmt.Sprintf("%sresults.txt", partial_path)
	}

	// path to write average and best times among all runs for each mode and data directory
	// averagesPath := fmt.Sprintf("%saverages.txt", partial_path)
	bestTotalTimesPath := fmt.Sprintf("%sbestTimes.txt", partial_path)
	bestParallTimesPath := fmt.Sprintf("%sbestParallTimes.txt", partial_path)
	imagesPartialPath := partial_path
	speedUpsPath := fmt.Sprintf("%sspeedups.txt", partial_path)

	// Parse results file, compute and save average times and speedups
	dataSets := ParseResults(resultsPath)
	// averagesElapsed := ComputeAverageTimes(dataSets)
	bestTotalTimes := ComputeBestTimes(dataSets, bestTotalTimesPath, bestParallTimesPath)
	speedups := ComputeSpeedups(bestTotalTimes, speedUpsPath)

	// Plot speedups for each mode
	// colors for the lines for each dataDir
	dataDirColors := map[string]color.RGBA{
		"small":   {R: 0, G: 255, B: 0, A: 255}, // green
		"mixture": {R: 0, G: 0, B: 255, A: 255}, // blue
		"big":     {R: 255, G: 0, B: 0, A: 255}, // red
	}

	for mode, data := range speedups {
		// create a new plot
		p := plot.New()
		
		// set the title and axis labels (obs: new lines and spaces for padding)
		p.Title.Text = fmt.Sprintf("\nEditor speedup graph (%s)", mode)
		p.X.Label.Text = "Number of Threads \n "
		p.Y.Label.Text = "\nSpeedup"

		// add space between the title and beginning of the plot
		p.Title.Padding = vg.Points(20)
		p.Title.TextStyle.Font.Size = vg.Points(15)

		// add space between the axes and the plot
		p.X.Label.Padding = vg.Points(5)
		p.Y.Label.Padding = vg.Points(5)

		// set grid lines
		grid := plotter.NewGrid()
		p.Add(grid)

		// force Y axis to show numbers in every tick
		p.Y.Tick.Marker = CustomYTicks{}

		// background color gray
		// p.BackgroundColor = color.RGBA{R: 225, G: 225, B: 225, A: 255}

		colorIndex := 0
		for dataDir, threadsData := range data {
			// sort thread counts in ascending order to pass to the graph
			keys := make([]int, 0, len(threadsData))
			for k := range threadsData {
				keys = append(keys, k)
			}
			// Sort the thread counts
			sort.Ints(keys)

			// Create the plotter.XYs struct using the sorted keys
			pts := make(plotter.XYs, len(keys))
			for i, k := range keys {
				pts[i].X = float64(k)
				pts[i].Y = threadsData[k]
			}

			// create a line for the dataDir
			line, _ := plotter.NewLine(pts)
			
			// line width and color
			line.LineStyle.Width = vg.Points(1)
			line.LineStyle.Color = dataDirColors[dataDir]
			
			// create markers for the dataDir line
			scatter, _ := plotter.NewScatter(pts)
			scatter.GlyphStyle.Color = dataDirColors[dataDir]
			scatter.GlyphStyle.Radius = vg.Points(2) // set the radius as per your requirement

			// add the line and the scatter to the plot
			p.Add(line, scatter) // adding scatter here

			// add a legend for the line
			p.Legend.Top = true
			p.Legend.Left = true
			p.Legend.Add(dataDir, line)

			// add some padding to the borders of the plot
			xmin, xmax := p.X.Min, p.X.Max
			ymin, ymax := p.Y.Min, p.Y.Max

			xpadding := (xmax - xmin) * 0.02 // 10% of range
			ypadding := (ymax - ymin) * 0.02 // 10% of range

			p.X.Min = xmin - xpadding
			p.X.Max = xmax + xpadding

			p.Y.Min = ymin - ypadding
			p.Y.Max = ymax + ypadding

			// force X axis to show all threads values
			threads := make([]int, 0)
			for k := range threadsData {
				threads = append(threads, k)
			}
			p.X.Tick.Marker = CustomXTicks{Threads: threads}

			// change the color for the next dataDir
			colorIndex++
		}

		// save plot to a PNG file
		if err := p.Save(6*vg.Inch, 6*vg.Inch, fmt.Sprintf("%sspeedup-%s.png", imagesPartialPath ,mode)); err != nil {
			panic(err)
		}
	}
}

