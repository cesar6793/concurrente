package main

import (
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// Calcula la distancia euclidiana entre dos puntos
func distance(a, b []float64) float64 {
	var sum float64
	for i := range a {
		sum += math.Pow(a[i]-b[i], 2)
	}
	return math.Sqrt(sum)
}

// Encuentra el centroide más cercano a un punto dado
func closestCentroid(point []float64, centroids [][]float64, wg *sync.WaitGroup, assignments chan<- int) {
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

// Actualiza los centroides basándose en las asignaciones de puntos
func updateCentroids(points [][]float64, assignments []int, k int, wg *sync.WaitGroup, newCentroids chan<- [][]float64) {
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

// Suma dos vectores
func addVectors(a, b []float64) []float64 {
	result := make([]float64, len(a))
	for i := range a {
		result[i] = a[i] + b[i]
	}
	return result
}

func main() {
	// URL del archivo CSV en formato RAW en GitHub
	url := "https://raw.githubusercontent.com/cesar6793/concurrente/main/datos.csv"

	// Realiza una solicitud GET para obtener el archivo CSV
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error al obtener el archivo:", err)
		return
	}
	defer resp.Body.Close()

	// Lee el contenido del archivo
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error al leer el contenido del archivo:", err)
		return
	}

	// Crea un lector CSV
	reader := csv.NewReader(strings.NewReader(string(body)))

	// Lee todos los registros del CSV
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Println("Error al leer el archivo CSV:", err)
		return
	}

	// Convierte los registros en puntos flotantes
	points := make([][]float64, len(records))
	for i, record := range records {
		points[i] = make([]float64, len(record))
		for j, value := range record {
			value = strings.TrimPrefix(value, "\ufeff") // Maneja posibles BOM
			point, err := strconv.ParseFloat(value, 64)
			if err != nil {
				fmt.Println("Error al convertir el valor:", err)
				return
			}
			points[i][j] = point
		}
	}

	k := 4 // Número de clusters

	// Inicializa los centroides como los primeros k puntos
	centroids := make([][]float64, k)
	copy(centroids, points[:k])

	assignments := make([]int, len(points))
	maxIterations := 100

	for iter := 0; iter < maxIterations; iter++ {
		var wg sync.WaitGroup
		assignmentsChan := make(chan int, len(points))
		newCentroidsChan := make(chan [][]float64, 1)

		// Asignar puntos a los clusters más cercanos concurrentemente
		wg.Add(len(points))
		for _, point := range points {
			go closestCentroid(point, centroids, &wg, assignmentsChan)
		}
		wg.Wait()
		close(assignmentsChan)

		// Recopilar asignaciones
		i := 0
		for assign := range assignmentsChan {
			assignments[i] = assign
			i++
		}

		// Calcular nuevos centroides concurrentemente
		wg.Add(1)
		go updateCentroids(points, assignments, k, &wg, newCentroidsChan)
		wg.Wait()
		newCentroids := <-newCentroidsChan

		// Verificar convergencia
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

	fmt.Println("Centroides:")
	for _, centroid := range centroids {
		fmt.Println(centroid)
	}

	fmt.Println("Asignaciones de puntos:")
	fmt.Println(assignments)
}
