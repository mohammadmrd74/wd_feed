package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/fsnotify/fsnotify"
)

const (
	logFilePath         = "./access.log"
	remoteServerAddress = "127.0.0.1"
	remoteServerPort    = "12345" // replace with the actual port number
	lastPositionFile    = "./last_position.txt"
)

func sendLogEntry(logEntry string) {
	conn, err := net.Dial("tcp", remoteServerAddress+":"+remoteServerPort)
	if err != nil {
		log.Println("Error connecting to the remote server:", err)
		return
	}
	defer conn.Close()

	_, err = conn.Write([]byte(logEntry))
	if err != nil {
		log.Println("Error sending log entry:", err)
	}
}

func monitorLogFile(startPosition int64) {
	file, err := os.Open(logFilePath)
	if err != nil {
		log.Fatal("Error opening log file:", err)
	}
	defer file.Close()

	_, err = file.Seek(startPosition, 0)
	if err != nil {
		log.Fatal("Error seeking to the last position:", err)
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		logEntry := scanner.Text()
		sendLogEntry(logEntry)
	}

	if err := scanner.Err(); err != nil {
		log.Fatal("Error reading log file:", err)
	}

	// Save the new position to the last position file
	newPosition, err := file.Seek(0, os.SEEK_CUR)
	if err != nil {
		log.Fatal("Error getting the current position:", err)
	}

	err = ioutil.WriteFile(lastPositionFile, []byte(strconv.FormatInt(newPosition, 10)), 0644)
	if err != nil {
		log.Fatal("Error writing to last position file:", err)
	}
}

func watchLogFile() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("Error creating file watcher:", err)
	}
	defer watcher.Close()

	err = watcher.Add(logFilePath)
	if err != nil {
		log.Fatal("Error adding file to watcher:", err)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				fmt.Println("File modified. Sending new entries.")
				lastPosition, err := getLastPosition()
				if err != nil {
					log.Fatal("Error getting last position:", err)
				}
				monitorLogFile(lastPosition)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("Error watching file:", err)
		}
	}
}

func getLastPosition() (int64, error) {
	data, err := ioutil.ReadFile(lastPositionFile)
	if err != nil {
		return 0, err
	}
	lastPosition, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return 0, err
	}
	return lastPosition, nil
}

func main() {
	go watchLogFile()

	// Run an initial check
	lastPosition, err := getLastPosition()
	if err != nil {
		log.Fatal("Error getting last position:", err)
	}
	monitorLogFile(lastPosition)

	// Keep the program running
	select {}
}
