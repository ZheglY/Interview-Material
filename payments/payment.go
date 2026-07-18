package payments

type PaymentMethod interface {
	Pay(usd int) int
	Cancel(id int)
}

type PaymentModule struct {
	paymentsInfo map[int]PaymentInfo
	paymentMethod PaymentMethod
}

func NewPaymentModule(paymentMethod PaymentMethod) *PaymentModule {
	return &PaymentModule{
		paymentsInfo: make(map[int]PaymentInfo),
		paymentMethod: paymentMethod,
	}
}


func (p PaymentModule) Pay(description string, usd int) int {
	id := p.paymentMethod.Pay(usd)

	info := PaymentInfo{
		Description: description,
		Usd: usd,
		Canceled: false,
	}

	p.paymentsInfo[id] = info
}

func (p PaymentModule) Cancel(id int) {
	info, ok := p.paymentsInfo[id]

	if !ok {
		return
	}

	p.paymentMethod.Cancel(id)
	info.Canceled = true
	p.paymentsInfo[id] = info
}

func (p PaymentModule) Info(id int) PaymentInfo {
	info, ok := p.paymentsInfo[id]

	if !ok {
		return PaymentInfo{}
	}

	return info
}

func (p *PaymentModule) AllInfo() map[int]PaymentInfo {
	
}