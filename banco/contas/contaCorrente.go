package hobobank

//Titular tipo para definir o titular de uma conta de banco
type Titular struct {
	Nome, CPF, Senha string
}

//ContaCorrente de um banco
type ContaCorrente struct {
	Titular                    Titular
	NumeroAgencia, NumeroConta string
	Saldo                      float64
}

//Sacar método para realizar um saque de uma conta corrente
func (c *ContaCorrente) Sacar(valorDoSaque float64) string {
	podeSacar := valorDoSaque > 0 && valorDoSaque <= c.Saldo
	if podeSacar {
		c.Saldo -= valorDoSaque
		return "Saque realizado com sucesso"
	}
	return "Saldo insuficiente"
}

//Depositar método para realizar o depósito de uma conta corrente
func (c *ContaCorrente) Depositar(valorDoDeposito float64) (string, float64) {
	if valorDoDeposito > 0 {
		c.Saldo += valorDoDeposito
		return "Depósito realizado com sucesso", c.Saldo
	}
	return "O valor do depósito deve ser maior que zero", c.Saldo
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

//Login método para logar um cliente
func (c *ContaCorrente) Login() bool {
	return true
}
