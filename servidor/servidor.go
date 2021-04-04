package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const tamanhoMaxMensagem = 512

//Servidor struct para definir um servidor TCP
type Servidor struct {
	clientes       map[net.Conn]bool     //clientes conectados no Servidor
	cadastrar      chan net.Conn         //canal para registrar um novo cliente
	descadastrar   chan net.Conn         //canal para cancelar o registro de um cliente que se desconectou
	sessoes        map[net.Conn]*Cliente //Jogos ativos
	iniciarSessao  chan *Cliente         //canal para iniciar uma nova sessão de um cliente conectado
	encerrarSessao chan *Cliente         //canal para encerrar a sessão de um cliente conectado
	proximaConta   int                   //próxima conta a ser atribuída pelo cliente
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
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	servidor := Servidor{
		clientes:       make(map[net.Conn]bool),
		cadastrar:      make(chan net.Conn),
		descadastrar:   make(chan net.Conn),
		sessoes:        make(map[net.Conn]*Cliente),
		iniciarSessao:  make(chan *Cliente),
		encerrarSessao: make(chan *Cliente),
		proximaConta:   r1.Intn(9999),
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
		case conta := <-servidor.iniciarSessao:
			servidor.sessoes[conta.Socket] = conta
			_ = salvarContaEmArquivo(conta.Conta)
			fmt.Println("Nova sessão iniciada.")
		//se houver um novo cliente no canal de encerrarJogo, vai retirar do mapa de jogos para fechar o jogo com ele
		case conta := <-servidor.encerrarSessao:
			_, existe := servidor.sessoes[conta.Socket]
			if existe {
				delete(servidor.sessoes, conta.Socket)
				fmt.Println("Uma sessão foi encerrada.")
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
		strMensagem := string(mensagem[:tamMensagem])
		if tamMensagem > 0 {
			comando := mensagem[0]
			switch comando {
			//comando para criar uma conta
			case '0':
				dadosNovaConta := strings.Split(strMensagem, ";")
				nome := dadosNovaConta[1]
				cpf := dadosNovaConta[2]
				senha := dadosNovaConta[3]
				numeroAgencia := "01"
				saldo := 0.0
				numeroConta := fmt.Sprintf("%04d", servidor.proximaConta)
				s1 := rand.NewSource(time.Now().UnixNano())
				r1 := rand.New(s1)
				servidor.proximaConta = r1.Intn(9999)

				conta := &ContaCorrente{
					Nome:          nome,
					CPF:           cpf,
					Senha:         senha,
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
				_, err = cliente.Write(mensagemAEnviar)
			//comando para logar no banco
			case '1':
				dadosLogin := strings.Split(strMensagem, ";")
				contaLogin := dadosLogin[1]
				agenciaLogin := dadosLogin[2]
				senhaLogin := dadosLogin[3]

				conta, err := lerContaDoArquivo(contaLogin)

				if err == nil && contaLogin == conta.NumeroConta && agenciaLogin == conta.NumeroAgencia && senhaLogin == conta.Senha {
					sessao := &Cliente{conta, cliente}
					mensagemAEnviar = []byte("S")
					servidor.iniciarSessao <- sessao
				} else {
					mensagemAEnviar = []byte("N")
				}
				cliente.Write(mensagemAEnviar)
			//comando para sacar dinheiro
			case '2':
				dadosSaque := strings.Split(strMensagem, ";")
				dinheiroASacar, err := strconv.ParseFloat(dadosSaque[1], 64)
				if err != nil {
					mensagemAEnviar = []byte("Erro ao converter valor a sacar")
				} else {
					conta, err := lerContaDoArquivo(servidor.sessoes[cliente].Conta.NumeroConta)

					if err != nil {
						mensagemAEnviar = []byte("Erro ao acessar conta de cliente")
					} else {
						if conta.Sacar(dinheiroASacar) {
							mensagemAEnviar = []byte("Saque realizado com sucesso")
						} else {
							mensagemAEnviar = []byte("Não conseguiu realizar o saque: saldo insuficiente")
						}
						salvarContaEmArquivo(conta)
					}
				}
				cliente.Write(mensagemAEnviar)
			//comando para depositar dinheiro
			case '3':
				dadosDeposito := strings.Split(strMensagem, ";")
				dinheiroADepositar, err := strconv.ParseFloat(dadosDeposito[1], 64)
				if err != nil {
					mensagemAEnviar = []byte("Erro ao converter valor a depositar")
				} else {
					conta, err := lerContaDoArquivo(servidor.sessoes[cliente].Conta.NumeroConta)

					if err != nil {
						mensagemAEnviar = []byte("Erro ao acessar conta de cliente")
					} else {
						if conta.Depositar(dinheiroADepositar) {
							mensagemAEnviar = []byte("Depósito realizado com sucesso")
						} else {
							mensagemAEnviar = []byte("Não conseguiu realizar o depósito: valor a depositar deve ser positivo")
						}
						salvarContaEmArquivo(conta)
					}
				}
				cliente.Write(mensagemAEnviar)
			//comando para transferir dinheiro
			case '4':
				dadosTransferencia := strings.Split(strMensagem, ";")
				contaATransferir := dadosTransferencia[1]
				dinheiroATransferir, _ := strconv.ParseFloat(dadosTransferencia[2], 64)

				conta, _ := lerContaDoArquivo(servidor.sessoes[cliente].Conta.NumeroConta)
				contaTransf, _ := lerContaDoArquivo(contaATransferir)

				if conta.Transferir(dinheiroATransferir, contaTransf) {
					mensagemAEnviar = []byte("Transferência realizada com sucesso")
				} else {
					mensagemAEnviar = []byte("Não conseguiu realizar a transferência: saldo insuficiente ou valor negativo")
				}
				salvarContaEmArquivo(conta)
				salvarContaEmArquivo(contaTransf)
				cliente.Write(mensagemAEnviar)
			//comando para imprimir saldo
			case '5':
				conta, _ := lerContaDoArquivo(servidor.sessoes[cliente].Conta.NumeroConta)
				salvarContaEmArquivo(conta)
				mensagemAEnviar = []byte("Saldo = R$" + fmt.Sprintf("%.2f", conta.Saldo))
				cliente.Write(mensagemAEnviar)
			//comando para encerrar a sessão com o cliente passado como parâmetro
			case '6':
				servidor.encerrarSessao <- servidor.sessoes[cliente]
				mensagemAEnviar = []byte("Sessão encerrada.")
				cliente.Write(mensagemAEnviar)
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
func lerContaDoArquivo(numConta string) (*ContaCorrente, error) {
	nomeArquivo := numConta + ".json"
	conta := &ContaCorrente{}

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
func salvarContaEmArquivo(conta *ContaCorrente) error {
	nomeArquivo := conta.NumeroConta + ".json"

	json, err := json.Marshal(conta)
	if err != nil {
		return errors.New("Erro ao converter a conta para json: " + err.Error())
	}

	arquivo, err := os.OpenFile(nomeArquivo, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		return errors.New("Erro ao salvar arquivo: " + err.Error())
	}

	n, err := arquivo.WriteString(string(json))
	if err != nil {
		fmt.Println("n ", n)
		fmt.Println("erro: ", err.Error())
	}

	err = arquivo.Close()
	if err != nil {
		fmt.Println("ERRO: ", err.Error())
	}
	return nil
}

type Cliente struct {
	Conta  *ContaCorrente
	Socket net.Conn
}

//ContaCorrente de um banco
type ContaCorrente struct {
	Nome, CPF, Senha           string
	NumeroAgencia, NumeroConta string
	Saldo                      float64
}

//Sacar método para realizar um saque de uma conta corrente
func (c *ContaCorrente) Sacar(valorDoSaque float64) bool {
	podeSacar := valorDoSaque > 0 && valorDoSaque <= c.Saldo
	if podeSacar {
		c.Saldo -= valorDoSaque
		return true
	}
	return false
}

//Depositar método para realizar o depósito de uma conta corrente
func (c *ContaCorrente) Depositar(valorDoDeposito float64) bool {
	if valorDoDeposito > 0 {
		c.Saldo += valorDoDeposito
		return true
	}
	return false
}

//Transferir método para realizar uma transferência
//entre esta conta corrente e a passada como parâmetro
func (c *ContaCorrente) Transferir(valorTransferencia float64, contaDestino *ContaCorrente) bool {
	if valorTransferencia > 0 && valorTransferencia <= c.Saldo {
		c.Saldo -= valorTransferencia
		contaDestino.Depositar(valorTransferencia)
		return true
	}
	return false
}

//ObterSaldo funçao para retornar o valor do Saldo
func (c *ContaCorrente) ObterSaldo() float64 {
	return c.Saldo
}
