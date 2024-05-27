package main

import (
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

func distance(a, b []float64) float64 {
	var sum float64
	for i := range a {
		sum += math.Pow(a[i]-b[i], 2)
	}
	return math.Sqrt(sum)
}

func closestCentroid(point []float64, centroids [][]float64, assignments chan<- int, wg *sync.WaitGroup) {
	defer wg.Done()
	minDist := math.MaxFloat64
	var closestIdx int
	for i, centroid := range centroids {
		dist := distance(point, centroid)
		if dist < minDist {
			minDist = dist
			closestIdx = i
		}
	}
	assignments <- closestIdx
}

func updateCentroids(points [][]float64, assignments []int, k int, newCentroids chan<- [][]float64, wg *sync.WaitGroup) {
	defer wg.Done()
	centroids := make([][]float64, k)
	counts := make([]int, k)

	for i := range centroids {
		centroids[i] = make([]float64, len(points[0]))
	}

	for i, point := range points {
		cluster := assignments[i]
		for j := range point {
			centroids[cluster][j] += point[j]
		}
		counts[cluster]++
	}

	for i := range centroids {
		for j := range centroids[i] {
			if counts[i] > 0 {
				centroids[i][j] /= float64(counts[i])
			}
		}
	}

	newCentroids <- centroids
}

func calculateCost(points [][]float64, centroids [][]float64, assignments []int) float64 {
	var totalCost float64
	for i, point := range points {
		centroid := centroids[assignments[i]]
		totalCost += distance(point, centroid)
	}
	return totalCost
}

func kMeans(points [][]float64, k int, maxIterations int) ([][]float64, []int, float64) {
	rand.Seed(time.Now().UnixNano())

	centroids := make([][]float64, k)
	for i := range centroids {
		centroids[i] = points[rand.Intn(len(points))]
	}

	assignments := make([]int, len(points))

	for iter := 0; iter < maxIterations; iter++ {
		var wg sync.WaitGroup
		assignmentsChan := make(chan int, len(points))
		newCentroidsChan := make(chan [][]float64, 1)

		wg.Add(len(points))
		for _, point := range points {
			go closestCentroid(point, centroids, assignmentsChan, &wg)
		}
		wg.Wait()
		close(assignmentsChan)

		i := 0
		for assign := range assignmentsChan {
			assignments[i] = assign
			i++
		}

		wg.Add(1)
		go updateCentroids(points, assignments, k, newCentroidsChan, &wg)
		wg.Wait()
		newCentroids := <-newCentroidsChan

		converged := true
		for i := range centroids {
			if distance(centroids[i], newCentroids[i]) > 0.001 {
				converged = false
				break
			}
		}
		if converged {
			break
		}

		centroids = newCentroids
	}

	cost := calculateCost(points, centroids, assignments)
	return centroids, assignments, cost
}

func main() {

	url := "https://raw.githubusercontent.com/cesar6793/concurrente/main/dataset.csv"

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error al obtener el archivo:", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error al leer el contenido del archivo:", err)
		return
	}

	reader := csv.NewReader(strings.NewReader(string(body)))

	records, err := reader.ReadAll()
	if err != nil {
		fmt.Println("Error al leer el archivo CSV:", err)
		return
	}

	points := make([][]float64, len(records))
	for i, record := range records {
		points[i] = make([]float64, len(record))
		for j, value := range record {
			value = strings.TrimPrefix(value, "\ufeff")
			point, err := strconv.ParseFloat(value, 64)
			if err != nil {
				fmt.Println("Error al convertir el valor:", err)
				return
			}
			points[i][j] = point
		}
	}

	k := 4
	maxIterations := 100
	bestCentroids := [][]float64{}
	bestAssignments := []int{}
	bestCost := math.MaxFloat64

	numRuns := 1000

	for run := 0; run < numRuns; run++ {
		centroids, assignments, cost := kMeans(points, k, maxIterations)
		if cost < bestCost {
			bestCost = cost
			bestCentroids = centroids
			bestAssignments = assignments
		}
		fmt.Printf("Run %d, Cost: %.4f\n", run+1, cost)
	}

	fmt.Println("Mejores centroides encontrados:")
	for _, centroid := range bestCentroids {
		fmt.Println(centroid)
	}

	fmt.Println("Mejores asignaciones de puntos:")
	fmt.Println(bestAssignments)
	fmt.Printf("Mejor costo: %.4f\n", bestCost)
}
