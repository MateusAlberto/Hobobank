package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

const tamanhoMaxMensagem = 512

var johnLennon *bufio.Reader

func main() {
	enderecoEPorta := os.Args[1]

	fmt.Println("Conectando no servidor", enderecoEPorta)
	//Conectando no endereço e porta especificados (pelo padrão endereço:porta)
	socket, err := net.Dial("tcp", enderecoEPorta)
	if err != nil {
		fmt.Println("Ocorreu um erro ao tentar se conectar ao servidor:", err)
		os.Exit(-1)
	}
	fmt.Println("Conexão realizada com sucesso")

	cliente := &Cliente{socket: socket}
	defer cliente.socket.Close()

	//johnLennon é um leitor
	johnLennon = bufio.NewReader(os.Stdin)

	//Loop do Menu Principal (se sair deste loop, é porque quis se desconectar do servidor)
	for {
		exibirMenuPrincipal()
		mensagem, err := johnLennon.ReadString('\n')
		if err == io.EOF {
			return //se deu EOF na leitura padrão, é porque o programa cliente foi fechado
		}
		mensagem = strings.Trim(mensagem, "\r\n")
		opcao := strings.ToUpper(strings.Trim(mensagem, " \r\n"))
		switch opcao {
		case "1":
			if cliente.logar() {
				cliente.gerenciarSessao()
			}
		case "2":
			cliente.criarConta()
		case "3":
			fmt.Println("Obrigado por acessar o HoboBank!")
			return //vai fechar o socket por causa do comando defer
		default:
			fmt.Print("Por favor, digite uma opção de menu válida.\n\n")
		}
	}

}

//Cliente struct que define um cliente para se conectar no servidor via TCP
type Cliente struct {
	socket      net.Conn
	numeroConta string
}

//Funçao que  vai ser responsável por receber os dados vindos do servidor
func (cliente *Cliente) receber() {
	mensagem := make([]byte, tamanhoMaxMensagem)
	tamMensagem, err := cliente.socket.Read(mensagem)
	//Se tiver algum erro, fecha a conexão
	if err != nil {
		fmt.Println("Ocorreu um erro de comunicação com o servidor:", err)
		cliente.socket.Close()
	}
	if tamMensagem > 0 {
		fmt.Println("\nServidor:", string(mensagem))
	}
}

//Função que lê os dados do cliente e cria uma conta para ele
func (cliente *Cliente) criarConta() {
	fmt.Print("\n------ CRIAR CONTA ------\n",
		"Seus dados:\n",
		"Nome: ")
	nome, _ := johnLennon.ReadString('\n')
	nome = strings.Trim(nome, "\r\n")
	fmt.Print("\nCPF: ")
	cpf, _ := johnLennon.ReadString('\n')
	cpf = strings.Trim(cpf, "\r\n")
	fmt.Print("\nSenha: ")
	senha, _ := johnLennon.ReadString('\n')
	senha = strings.Trim(senha, "\r\n")
	mensagemAEnviar := make([]byte, tamanhoMaxMensagem)
	mensagemAEnviar = []byte("0;" + nome + ";" + cpf + ";" + senha)
	cliente.socket.Write(mensagemAEnviar)
	mensagemAReceber := make([]byte, tamanhoMaxMensagem)
	tamMensagemAReceber, err := cliente.socket.Read(mensagemAReceber)

	//Se tiver algum erro, fecha a conexão
	if err != nil {
		fmt.Println("Ocorreu um erro de comunicação com o servidor:", err, tamMensagemAReceber)
		cliente.socket.Close()
	}
	if tamMensagemAReceber > 0 {
		if strings.Compare(strings.Trim(string(mensagemAReceber), "\r\n"), "N") != 0 {
			fmt.Println(string(mensagemAReceber))
			cliente.numeroConta = strings.Split(string(mensagemAReceber), ";")[1]
			fmt.Print("Conta criada com sucesso! Seu número de conta é " + cliente.numeroConta + " e o de agência é 01.\n\n")
		} else {
			fmt.Println("Não conseguiu criar a conta")
		}
	}
}

//Função para logar o cliente na conta
func (cliente *Cliente) logar() bool {
	logado := false
	fmt.Print("\n------ LOGIN ------\n",
		"Número da conta: ")
	conta, _ := johnLennon.ReadString('\n')
	conta = strings.Trim(conta, "\r\n")
	fmt.Print("\nAgência: ")
	agencia, _ := johnLennon.ReadString('\n')
	agencia = strings.Trim(agencia, "\r\n")
	fmt.Print("\nSenha: ")
	senha, _ := johnLennon.ReadString('\n')
	senha = strings.Trim(senha, "\r\n")
	mensagemAEnviar := make([]byte, tamanhoMaxMensagem)
	mensagemAEnviar = []byte("1;" + conta + ";" + agencia + ";" + senha)
	cliente.socket.Write(mensagemAEnviar)
	mensagemAReceber := make([]byte, tamanhoMaxMensagem)
	tamMensagemAReceber, err := cliente.socket.Read(mensagemAReceber)

	//Se tiver algum erro, fecha a conexão
	if err != nil {
		fmt.Println("Ocorreu um erro de comunicação com o servidor:", err)
		cliente.socket.Close()
	}
	if tamMensagemAReceber > 0 {
		if string(mensagemAReceber[:tamMensagemAReceber]) == "S" {
			logado = true
			fmt.Print("Logado com sucesso!\n\n")
			cliente.numeroConta = conta
		}
	}
	return logado
}

//Função que vai lidar com a sessão
func (cliente *Cliente) gerenciarSessao() {
	mensagemAEnviar := make([]byte, tamanhoMaxMensagem)

	for {
		zerarBuffer(mensagemAEnviar)
		exibirMenuBanco()
		mensagem, _ := johnLennon.ReadString('\n')
		mensagem = strings.Trim(strings.ToUpper(mensagem), " \r\n")
		switch mensagem {
		case "1":
			fmt.Print("\nDigite a quantidade a sacar: ")
			dinheiroASacar, _ := johnLennon.ReadString('\n')
			dinheiroASacar = strings.Trim(strings.ToUpper(dinheiroASacar), " \r\n")
			mensagemAEnviar = []byte("2;" + dinheiroASacar)
			cliente.socket.Write(mensagemAEnviar)
			cliente.receber()
		case "2":
			fmt.Print("\nDigite a quantidade a depositar: ")
			dinheiroADepositar, _ := johnLennon.ReadString('\n')
			dinheiroADepositar = strings.Trim(strings.ToUpper(dinheiroADepositar), " \r\n")
			mensagemAEnviar = []byte("3;" + dinheiroADepositar)
			cliente.socket.Write(mensagemAEnviar)
			cliente.receber()
		case "3":
			fmt.Print("\nDigite o número da conta a transferir: ")
			contaATransferir, _ := johnLennon.ReadString('\n')
			contaATransferir = strings.Trim(strings.ToUpper(contaATransferir), " \r\n")
			fmt.Print("\nDigite a quantidade a transferir: ")
			dinheiroATransferir, _ := johnLennon.ReadString('\n')
			dinheiroATransferir = strings.Trim(strings.ToUpper(dinheiroATransferir), " \r\n")
			mensagemAEnviar = []byte("4;" + contaATransferir + ";" + dinheiroATransferir)
			cliente.socket.Write(mensagemAEnviar)
			cliente.receber()
		case "4":
			mensagemAEnviar = []byte("5")
			cliente.socket.Write(mensagemAEnviar)
			cliente.receber()
		case "5":
			fmt.Print("\nSaindo do HoboBank...\n\n")
			mensagemAEnviar = []byte("6")
			cliente.socket.Write(mensagemAEnviar)
			cliente.receber()
			return
		default:
			fmt.Println("\nPor favor, digite um comando correto.")
		}
	}
}

//Exibe o menu principal
func exibirMenuPrincipal() {
	fmt.Print("\n------ MENU PRINCIPAL ------\n",
		"Digite os seguintes comandos:\n",
		"1 - Login\n",
		"2 - Criar Conta\n",
		"3 - Desconectar\n\n",
		"Digite sua opção: ")
}

//Exibe o menu de um usuário logado
func exibirMenuBanco() {
	fmt.Print("\n------ MENU HOBOBANK ------\n",
		"Digite sua opção:\n",
		"1 - Sacar dinheiro\n",
		"2 - Depositar dinheiro\n",
		"3 - Transferir dinheiro\n",
		"4 - Imprimir saldo\n",
		"5 - Sair do Hobobank\n\n",
		"Digite sua opção: ")
}

func sacar() {

}

//Pequena função para zerar o buffer
func zerarBuffer(array []byte) {
	for i := 0; i < len(array); i++ {
		array[i] = 0
	}
}
