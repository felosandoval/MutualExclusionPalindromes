package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Node struct {
	nodeID     int
	nodeStatus string
	timeCount  int
}

type Msg struct {
	timeCount int
	senderID  int
}

type Palindrome struct {
	node       *Node
	nodeStatus string
	timeCount  int
	wg         *sync.WaitGroup
	fileName   string
	rows, cols int
}

var mutex sync.Mutex
var allFound bool
var foundProcesses = make(map[int]bool)
var requestQueues = make(map[int][]Msg)
var nodes []*Node

func (p *Palindrome) palindromeSearch() {
	defer p.wg.Done()

	for {
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)))

		mutex.Lock()
		if allFound {
			mutex.Unlock()
			break
		}

		if p.nodeStatus == "IDLE" {
			p.timeCount += 1

			for j := 0; j < len(nodes); j++ {
				if j != p.node.nodeID-1 {
					message := Msg{timeCount: p.timeCount, senderID: p.node.nodeID}
					requestQueues[j+1] = append(requestQueues[j+1], message)
					sendMsg(j+1, message)
				}
			}

			p.nodeStatus = "REQUESTING"
		}

		file, err := os.OpenFile(p.fileName, os.O_RDWR, 0644)
		if err != nil {
			fmt.Printf("Proceso %d obtuvo un error al abrir el archivo: %v\n", p.node.nodeID, err)
			mutex.Unlock()
			return
		}

		scanner := bufio.NewScanner(file)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}

		palindromeFound := false
		for i, line := range lines {
			if isPalindrome(line) {
				fmt.Printf("Proceso %d encontró el palindromo: %s\n", p.node.nodeID, strings.Replace(line, " ", "", -1))
				lines[i] = strings.Repeat(strconv.Itoa(p.node.nodeID)+" ", len(strings.Replace(line, " ", "", -1)))
				foundProcesses[p.node.nodeID] = true
				file.Seek(0, 0)
				for _, line := range lines {
					_, err := file.WriteString(line + "\n")
					if err != nil {
						fmt.Printf("Proceso %d obtuvo un error al escribir en el archivo: %v\n", p.node.nodeID, err)
					}
				}
				palindromeFound = true
				break
			}
		}

		file.Close()
		mutex.Unlock()

		if palindromeFound {
			continue
		}

		mutex.Lock()
		if !palindromeFound {
			allFound = true
		}
		mutex.Unlock()
	}

	if p.nodeStatus == "IDLE" {
		for _, message := range requestQueues[p.node.nodeID] {
			sendReply(message.senderID, p.node.nodeID)
		}
		requestQueues[p.node.nodeID] = []Msg{}
	}
}

func sendMsg(receiverID int, message Msg) {
	receiver := nodes[receiverID-1]
	//fmt.Printf("Proceso %d envió un mensaje a proceso %d: %+v\n", message.senderID, receiverID, message)
	receiver.handleMsg(message)
}

func sendReply(receiverID int, senderID int) {
	receiver := nodes[receiverID-1]
	message := Msg{senderID: senderID}
	//fmt.Printf("Proceso %d envió REPLY a proceso %d\n", senderID, receiverID)
	receiver.handleMsg(message)
}

func (n *Node) handleMsg(message Msg) {
	switch n.nodeStatus {
	case "REQUESTING":
		if message.timeCount < n.timeCount || (message.timeCount == n.timeCount && message.senderID < n.nodeID) {
			sendReply(message.senderID, n.nodeID)
		} else {
			requestQueues[n.nodeID] = append(requestQueues[n.nodeID], message)
		}
	case "EXECUTING":
		sendReply(message.senderID, n.nodeID)
	default:
		//fmt.Printf("Proceso %d recibio mensaje: %+v\n", n.nodeID, message)
	}
}

func isPalindrome(s string) bool {
    str := strings.Replace(s, " ", "", -1)

    if str == "" {
        return false
    }

    isNumericOnly := true
    for _, c := range str {
        if !strings.ContainsRune("0123456789", c) {
            isNumericOnly = false
            break
        }
    }

    if isNumericOnly {
        return false
    }

    for i := 0; i < len(str)/2; i++ {
        if str[i] != str[len(str)-i-1] {
            return false
        }
    }

    return true
}

func main() {
	rand.Seed(time.Now().UnixNano())

	if len(os.Args) != 5 {
		fmt.Println("Usa: ./main <numProcesos> <numFilas> <numColumnas> <nombreArchivo>")
		return
	}

	// 1 ARGUMENTO (NUMERO DE PROCESOS)
	numNodes, err := strconv.Atoi(os.Args[1])
	if err != nil {
		return
	}

	nodes = make([]*Node, numNodes)
	for i := range nodes {
		nodes[i] = &Node{nodeID: i + 1, nodeStatus: "IDLE"}
	}

	// 2 ARGUMENTO (FILAS)
	rows, err := strconv.Atoi(os.Args[2])
	if err != nil {
		return
	}

	// 3 ARGUMENTO (COLUMNAS)
	cols, err := strconv.Atoi(os.Args[3])
	if err != nil {
		return
	}

	// 4 ARGUMENTO (NOMBRE DE ARCHIVO CON LA MATRIZ)
	fileName := os.Args[4]

	var wg sync.WaitGroup

	// SE INSTANCIAN LOS PROCESOS CON ATRIBUTOS POR DEFECTO
	for _, node := range nodes {
		wg.Add(1)
		p := &Palindrome{node: node, nodeStatus: "IDLE", fileName: fileName, rows: rows, cols: cols, wg: &wg}
		// SE MANDAN A LOS PROCESOS A BUSCAR LOS PALINDROMOS
		go p.palindromeSearch()
	}
	wg.Wait()

	// SE IMPRIMEN PROCESOS QUE NO ENCONTRARON PALINDROMOS
	for _, node := range nodes {
		if !foundProcesses[node.nodeID] {
			fmt.Printf("Proceso %d no encontró ningún palindromo\n", node.nodeID)
		}
	}
}