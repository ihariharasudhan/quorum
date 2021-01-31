package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/rpc"
	"os"
	"time"
)

//structure that defines the type that will be used for communication
type FileDir struct {
	Data string
}

type Vote struct {
	ProcessID string
	VoteSend  string
}

var readBlocked, writeBlocked bool
var count int

func OkServer() {

	s, err := net.ResolveUDPAddr("udp4", "127.0.0.1:49202")
	if err != nil {
		fmt.Println(err)
		return
	}

	connection, err := net.ListenUDP("udp4", s)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer connection.Close()
	buffer := make([]byte, 1024)
	for {
		size, _, _ := connection.ReadFromUDP(buffer)
		if size == 0 {
			fmt.Println("Error")
		}
		writeBlocked = false
		readBlocked = false
	}
}

//function that will be called remotely
//the function gets the full path of the file as the input and returns the contents of the file as the output
func (f *FileDir) GetFilesContents(file []byte, reply *FileDir) error {

	//line contains full path of the file
	time.Sleep(5 * time.Second)
	filePath := string(file) //taking the path of the file from the byte variable

	content, err := ioutil.ReadFile(filePath) //reading the contents of the file
	if err != nil {
		fmt.Println("File reading error", err)
		return nil
	}

	data := string(content) //converting the contents of the file to string
	*reply = FileDir{data}  //referencing the content to the sent to the client
	return nil
}

func (f *FileDir) WriteContent(file []byte, reply *FileDir) error {

	var result FileDir

	fmt.Println("Entered RPC write")

	err := ioutil.WriteFile("client2.txt", file, 0644)
	if err != nil {
		fmt.Println(err)
		result = FileDir{"Fail"}
	} else {
		fmt.Println("Content changed")
		result = FileDir{"Success"}
	}

	fmt.Println(writeBlocked)
	*reply = result
	return nil
}

func fileRead(IP string) {

	client, err := rpc.Dial("tcp", IP) //connecting with the server
	if err != nil {
		log.Fatal(err)
	}
	var content FileDir

	filePath := []byte("client2.txt")
	err = client.Call("FileDir.GetFilesContents", filePath, &content) //asynchronous RPC for the function that returns the contents of the file
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("The contents of the file are \n", content.Data) //printing the contents of the file
}

func serverRunning() {

	address, err := net.ResolveTCPAddr("tcp", "127.0.0.1:12501") //local loopback address with port number with TCP as the transport layer protocol
	if err != nil {
		log.Fatal(err)
	}

	inbound, err := net.ListenTCP("tcp", address) //listening to the TCP port
	if err != nil {
		log.Fatal(err)
	}

	forFiles := new(FileDir)
	rpc.Register(forFiles) //exporting the functions of the struct FileDir
	rpc.Accept(inbound)    //accepting connections from the clients
}

func writeContent(IP string, fileContent []byte) {

	client, err := rpc.Dial("tcp", IP) //connecting with the server
	if err != nil {
		log.Fatal(err)
	}
	var content FileDir

	fmt.Println("In the local write function")

	err = client.Call("FileDir.WriteFile", fileContent, &content) //asynchronous RPC for the function that returns the contents of the file
	if err != nil {
		log.Fatal(err)
		fmt.Println("RPC called failed")
	}

	if content.Data == "Success" {
		fmt.Println("Write successful")
	} else if content.Data == "Fail" {
		fmt.Println("Write Failed")
	}

}

func (f *Vote) VoteReply(operation []byte, reply *Vote) error {

	op := string(operation)
	var voteS Vote

	fmt.Println(readBlocked, writeBlocked)

	if op == "r" {
		fmt.Println("Got read request")
		if writeBlocked == false {
			voteS.ProcessID = "2"
			voteS.VoteSend = "Yes"
			readBlocked = true
		} else {
			voteS.ProcessID = "2"
			voteS.VoteSend = "No"
		}
	}
	if op == "w" {
		fmt.Println("Got write request")
		if writeBlocked == false && readBlocked == false {
			voteS.ProcessID = "2"
			voteS.VoteSend = "Yes"
			writeBlocked = true
		} else {
			voteS.ProcessID = "2"
			voteS.VoteSend = "No"
		}

	}
	*reply = voteS
	return nil
}

func getVote(IP string, mode string) bool {

	var voteGot Vote
	var toMain bool
	client, err := rpc.Dial("tcp", IP) //connecting with the server
	if err != nil {
		log.Fatal(err)
	}

	err = client.Call("Vote.VoteReply", []byte(mode), &voteGot) //asynchronous RPC for the function that returns the contents of the file
	if err != nil {
		log.Fatal(err)
	}

	if voteGot.VoteSend == "Yes" {
		count = count + 1
		toMain = true
	} else {
		toMain = false
	}
	printResult := "\n Got " + voteGot.VoteSend + " from " + voteGot.ProcessID
	fmt.Println(printResult)
	return toMain
}

func voterRunning() {
	address, err := net.ResolveTCPAddr("tcp", "127.0.0.1:12601") //local loopback address with port number with TCP as the transport layer protocol
	if err != nil {
		log.Fatal(err)
	}

	inbound, err := net.ListenTCP("tcp", address) //listening to the TCP port
	if err != nil {
		log.Fatal(err)
	}

	forVotes := new(Vote)
	rpc.Register(forVotes) //exporting the functions of the struct FileDir
	rpc.Accept(inbound)    //accepting connections from the clients

}

func sendAck(IP string) {

	var voteR Vote
	voteR.ProcessID = ""
	voteR.VoteSend = "Done"

	voteSend, _ := json.Marshal(voteR)
	setUp, err := net.ResolveUDPAddr("udp4", IP)
	connection, err := net.DialUDP("udp4", nil, setUp)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer connection.Close()
	_, err = connection.Write(voteSend)

}

func main() {

	go serverRunning()
	go voterRunning()
	go OkServer()

	readBlocked = false
	writeBlocked = false
	count = 0
	var option int
	allIPsForVote := [3]string{"127.0.0.1:12600", "127.0.0.1:12602", "127.0.0.1:12603"}
	allIPsForWrite := [3]string{"127.0.0.1:12500", "127.0.0.1:12502", "127.0.0.1:12503"}
	allIPsForAck := [3]string{"127.0.0.1:49204", "127.0.0.1:49203", "127.0.0.1:49201"}
	for {

		fmt.Println("\nEnter the operation that you want to perform\n1.File write\n2.File read\n3.Exit")
		fmt.Scanln(&option)

		if option == 1 {
			if writeBlocked == false {
				fmt.Println("\nEnter the content to add: ") //getting the file path
				in := bufio.NewReader(os.Stdin)
				fileContent, _, err := in.ReadLine()
				if err != nil {
					log.Fatal(err)
				}
				Content := string(fileContent)
				fileContent = []byte(Content + "\n")
				writeBlocked = true
				for i := 0; i < len(allIPsForVote); i++ {
					result := getVote(allIPsForVote[i], "w")
					if result != true {
						continue
					}
				}
				if count == 3 {
					for i := 0; i < len(allIPsForWrite); i++ {
						writeContent(allIPsForWrite[i], fileContent)
					}
					for i := 0; i < len(allIPsForAck); i++ {
						sendAck(allIPsForAck[i])
					}
				}
				err = ioutil.WriteFile("client2.txt", fileContent, 0644)
				if err != nil {
					fmt.Println(err)
				}
				writeBlocked = false
				count = 0
			} else {
				fmt.Println("Another process writing/reading")
			}
		} else if option == 2 {
			if writeBlocked == false {
				for i := 0; i < len(allIPsForVote); i++ {
					result := getVote(allIPsForVote[i], "r")
					if result != true {
						continue
					}
				}
				readBlocked = true
				if count == 3 {
					for i := 0; i < len(allIPsForWrite); i++ {
						fileRead(allIPsForWrite[i])
					}
					for i := 0; i < len(allIPsForAck); i++ {
						sendAck(allIPsForAck[i])
					}
				}
				readBlocked = false
			} else {
				fmt.Println("Another process writing")
			}
		} else if option == 3 {
			break
		}
	}

}
