package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows/registry"
)

type Stat struct {
	SessionCount int64 `json:"sessionCount"`
	TotalMsecs   int64 `json:"totalMsecs"`
}

type PerGameStats map[string]Stat

type PerServerStats map[string]struct {
	TotalStat    Stat         `json:"totalStat"`
	PerGameStats PerGameStats `json:"perGameStats"`
}

type MonthStat struct {
	TotalStat      Stat           `json:"totalStat"`
	PerServerStats PerServerStats `json:"perServerStats"`
	PerGameStats   PerGameStats   `json:"perGameStats"`
}

type Data struct {
	MonthStat MonthStat `json:"monthStat"`
}

func main() {
	var x int64
	var secInMsec int64 = 1000
	var hoursInSecond int64 = 3600
	var minutesInSec int64 = 60

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal("Ошибка получения текущей деректории: ", err)
	}

	log.SetOutput(logFile(dir))

	gameID()

	currentTime := time.Now()
	csvFile, err := os.Create("game30day - " + currentTime.Format("02012006-150405") + ".csv")
	if err != nil {
		log.Fatal("Failed to create file: ", err)
	}
	defer csvFile.Close()
	writer := csv.NewWriter(csvFile)
	defer writer.Flush()
	writer.Write([]string{"Игра", "Количество сессий", "Продолжительность"})

	authToken := getToken()

	urlStat := "https://services.drova.io/accounting/statistics/myserverusageprepared"
	responseString, err := getFromURL(urlStat, authToken)
	if err != nil {
		log.Fatal(err)
	}
	var data Data
	json.Unmarshal([]byte(responseString), &data)

	for idGame, stats := range data.MonthStat.PerGameStats {
		fmt.Printf("ID: %s, SessionCount: %d, TotalMsecs: %d\n", idGame, stats.SessionCount, stats.TotalMsecs)
		gameName, _ := keyValFile(idGame, "gamesID.txt")
		x = stats.TotalMsecs / secInMsec
		hours := x / hoursInSecond
		minutes := (x - (hours * hoursInSecond)) / minutesInSec
		seconds := x - (hours * hoursInSecond) - (minutes * minutesInSec)
		duration := fmt.Sprintf("%d:%d:%d", hours, minutes, seconds)
		sessionCount := fmt.Sprint(stats.SessionCount)
		writer.Write([]string{gameName, sessionCount, duration})
	}
}

func logFile(dir string) *os.File {
	logFilePath := "errors.log"
	logFilePath = filepath.Join(dir, logFilePath)

	logFile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal("Ошибка открытия файла", err)
	}
	defer logFile.Close()

	return logFile
}

func getToken() (authToken string) {
	regFolder := `SOFTWARE\ITKey\Esme`
	serverID := regGet(regFolder, "last_server") // получаем ID сервера
	regFolder += `\servers\` + serverID
	authToken = regGet(regFolder, "auth_token") // получаем токен для авторизации
	return
}

func regGet(regFolder, keys string) string {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, regFolder, registry.QUERY_VALUE)
	if err != nil {
		log.Printf("Failed to open registry key: %v\n", err)
	}
	defer key.Close()

	value, _, err := key.GetStringValue(keys)
	if err != nil {
		log.Printf("Failed to read last_server value: %v\n", err)
	}

	return value
}

func getFromURL(url, authToken string) (responseString string, err error) {
	_, err = http.Get("https://services.drova.io")
	if err != nil {
		log.Println("[ERROR] Сайт https://services.drova.io недоступен")
		return
	} else {
		client := &http.Client{}

		var resp *http.Response

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Println("[ERROR] Ошибка создания запроса: ", err)
			return "", err
		}

		q := req.URL.Query()
		req.URL.RawQuery = q.Encode()

		req.Header.Set("X-Auth-Token", authToken)

		resp, err = client.Do(req)
		if err != nil {
			log.Println("[ERROR] Ошибка отправки запроса: ", err)
			return "", err
		}
		defer resp.Body.Close()
		var buf bytes.Buffer
		_, err = io.Copy(&buf, resp.Body)
		if err != nil {
			log.Println("[ERROR] Ошибка записи запроса в буфер: ", err)
			return "", err
		}

		responseString = buf.String()
	}

	return responseString, err
}

type GameNameID struct {
	ProductID string `json:"productId"`
	Title     string `json:"title"`
}

func gameID() {
	gameIDforName := "gamesID.txt"
	// gameIDforName := filepath.Join(dir, fileGameID)
	resp, err := http.Get("https://services.drova.io/product-manager/product/listfull2")
	if err != nil {
		fmt.Println("Ошибка при выполнении запроса:", err)
		return
	}
	defer resp.Body.Close()

	var products []GameNameID
	err = json.NewDecoder(resp.Body).Decode(&products)
	if err != nil {
		fmt.Println("Ошибка при разборе JSON-ответа:", err)
		return
	}

	file, err := os.Create(gameIDforName)
	if err != nil {
		fmt.Println("Ошибка при создании файла:", err)
		return
	}
	defer file.Close()

	for _, product := range products {
		line := fmt.Sprintf("%s = %s\n", product.ProductID, product.Title)
		_, err = io.WriteString(file, line)
		if err != nil {
			fmt.Println("Ошибка при записи данных в файл:", err)
			return
		}
	}
	time.Sleep(1 * time.Second)
}

func keyValFile(keys, fileSt string) (string, error) {
	var val string
	file, err := os.Open(fileSt)
	if err != nil {
		fmt.Println("Ошибка при открытии файла:", err)
		return "Ошибка при открытии файла:", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	data := make(map[string]string)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, " = ")
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]
			data[key] = value
		}
	}

	if value, ok := data[keys]; ok {
		val = value

	} else {
		val = keys
	}
	return val, err
}

// func makeCSV() (csvFile *os.File) {
// 	currentTime := time.Now()
// 	csvFile, err := os.Create("game30day - " + currentTime.Format("02012006-150405") + ".csv")
// 	if err != nil {
// 		log.Fatal("Failed to create file: ", err)
// 	}
// 	defer csvFile.Close()

// 	return
// }
