package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	hobobank "trabLRSO/banco/contas"
)

const tamanhoMaxMensagem = 512

//Servidor struct para definir um servidor TCP
type Servidor struct {
	clientes       map[net.Conn]bool                    //clientes conectados no Servidor
	cadastrar      chan net.Conn                        //canal para registrar um novo cliente
	descadastrar   chan net.Conn                        //canal para cancelar o registro de um cliente que se desconectou
	sessoes        map[net.Conn]*hobobank.ContaCorrente //Jogos ativos
	iniciarSessao  chan net.Conn                        //canal para iniciar um novo jogo de um cliente
	encerrarSessao chan net.Conn                        //canal para encerrar um jogo com um cliente
	proximaConta   int                                  //próxima conta a ser atribuída pelo cliente
}

func main() {
	porta := os.Args[1]

	listener, err := net.Listen("tcp", ":"+porta)
	if err != nil {
		fmt.Println("Ocorreu um erro ao ouvir a porta:", err)
		os.Exit(-1)
	}
	fmt.Println("Servidor ouvindo na porta", porta)
	defer listener.Close() //vai garantir que irá fechar o listener assim que fechar o programa

	servidor := Servidor{
		clientes:       make(map[net.Conn]bool),
		cadastrar:      make(chan net.Conn),
		descadastrar:   make(chan net.Conn),
		sessoes:        make(map[net.Conn]*hobobank.ContaCorrente),
		iniciarSessao:  make(chan net.Conn),
		encerrarSessao: make(chan net.Conn),
		proximaConta:   1,
	}

	go servidor.iniciar()
	for {
		socket, err := listener.Accept()
		if err != nil {
			fmt.Println("Ocorreu um erro ao tentar conectar com um cliente:", err)
		}
		servidor.cadastrar <- socket
		go servidor.receber(socket)
	}
}

//Funcão que irá iniciar o cadastro e o descadastro dos clientes (acontece em paralelo por uma goroutine)
func (servidor *Servidor) iniciar() {
	for {
		select {
		//se houver um cliente novo no canal de cadastro, vai adicionar isso no mapa de clientes
		case socket := <-servidor.cadastrar:
			servidor.clientes[socket] = true
			fmt.Println("Novo cliente conectado.")
		//se houver um cliente no canal de descadastro, vai retirar do mapa e fechar a conexão com o cliente
		case socket := <-servidor.descadastrar:
			_, existe := servidor.clientes[socket]
			if existe {
				delete(servidor.clientes, socket)
				fmt.Println("Um cliente foi desconectado.")
			}
		//se houver um novo cliente no canal de iniciarSessao, vai adicionar no mapa de jogos e iniciar um novo jogo com ele
		case socket := <-servidor.iniciarSessao:
			servidor.sessoes[socket] = &hobobank.ContaCorrente{}
			if servidor.sessoes[socket].Login() {
				fmt.Println("Nova sessão iniciada.")
			}
		//se houver um novo cliente no canal de encerrarJogo, vai retirar do mapa de jogos para fechar o jogo com ele
		case socket := <-servidor.encerrarSessao:
			_, existe := servidor.sessoes[socket]
			if existe {
				delete(servidor.sessoes, socket)
				fmt.Println("Um jogo foi encerrado.")
			}
		}
	}
}

//Função que acontecerá o tempo todo em paralelo e será responsável por receber as mensagens dos clientes
func (servidor *Servidor) receber(cliente net.Conn) {
	mensagem := make([]byte, tamanhoMaxMensagem)
	mensagemAEnviar := make([]byte, tamanhoMaxMensagem)
	for {
		zerarBuffer(mensagem)
		tamMensagem, err := cliente.Read(mensagem)
		if err != nil {
			servidor.descadastrar <- cliente
			cliente.Close()
			break
		}
		fmt.Println(string(mensagem[:tamMensagem]))
		if tamMensagem > 0 {
			comando := mensagem[0]
			switch comando {
			//comando para logar no banco
			case '0':
				dadosLogin := strings.Split(string(mensagem), ";")
				contaLogin := dadosLogin[1]
				agenciaLogin := dadosLogin[2]
				senhaLogin := dadosLogin[3]

				conta, err := lerContaDoArquivo(contaLogin)

				if err == nil && contaLogin == conta.NumeroConta && agenciaLogin == conta.NumeroAgencia && senhaLogin == conta.Titular.Senha {
					mensagemAEnviar = []byte("S")
				} else {
					mensagemAEnviar = []byte("N")
				}
				cliente.Write(mensagemAEnviar)
			//comando para criar uma conta
			case '1':
				dadosNovaConta := strings.Split(string(mensagem), ";")
				nome := dadosNovaConta[1]
				cpf := dadosNovaConta[2]
				senha := dadosNovaConta[3]
				numeroAgencia := "01"
				saldo := 0.0
				numeroConta := fmt.Sprintf("%8d", servidor.proximaConta)
				servidor.proximaConta++

				conta := &hobobank.ContaCorrente{
					Titular: hobobank.Titular{
						Nome:  nome,
						CPF:   cpf,
						Senha: senha,
					},
					NumeroAgencia: numeroAgencia,
					NumeroConta:   numeroConta,
					Saldo:         saldo,
				}

				err := salvarContaEmArquivo(conta)

				if err != nil {
					mensagemAEnviar = []byte("N")
				} else {
					mensagemAEnviar = []byte("S;" + numeroConta)
				}
				cliente.Write(mensagemAEnviar)
			//comando para sacar dinheiro
			case '2':
				dadosSaque := strings.Split(string(mensagem), ";")
				dinheiroASacar := dadosSaque[1]
			//comando para depositar dinheiro
			case '3':

			//comando para transferir dinheiro
			case '4':

			//comando para imprimir saldo
			case '5':

			//comando para encerrar a sessão com o cliente passado como parâmetro
			case '6':
			}
		}
	}
}

//Pequena função para zerar o buffer
func zerarBuffer(array []byte) {
	for i := 0; i < len(array); i++ {
		array[i] = 0
	}
}

//Função que retorna se um arquivo existe ou não
func arquivoExiste(nomeArquivo string) bool {
	if _, err := os.Stat(nomeArquivo); err == nil {
		return true
	} else {
		return false
	}
}

//Lê uma conta de um arquivo, de acordo como especificado, e retorna a conta
func lerContaDoArquivo(numConta string) (*hobobank.ContaCorrente, error) {
	nomeArquivo := "/contas/" + numConta + ".json"
	conta := &hobobank.ContaCorrente{}

	if !arquivoExiste(nomeArquivo) {
		return nil, errors.New("Arquivo não existe")
	}

	arquivo, err := ioutil.ReadFile(nomeArquivo)

	if err != nil {
		return nil, errors.New("Erro ao ler o arquivo: " + err.Error())
	}

	err = json.Unmarshal(arquivo, conta)
	if err != nil {
		return nil, errors.New("Erro ao converter o json: " + err.Error())
	}

	return conta, nil
}

//Lê uma conta de um arquivo, de acordo como especificado, e retorna a conta
func salvarContaEmArquivo(conta *hobobank.ContaCorrente) error {
	nomeArquivo := "/contas/" + conta.NumeroConta + ".json"

	json, err := json.Marshal(conta)
	if err != nil {
		return errors.New("Erro ao converter a conta para json: " + err.Error())
	}

	err = ioutil.WriteFile(nomeArquivo, json, 0644)
	if err != nil {
		return errors.New("Erro ao salvar arquivo: " + err.Error())
	}
	return nil
}
