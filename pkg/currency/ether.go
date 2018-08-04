package currency

type Milliether float64

func (m Milliether) Ether() Ether {
	return Ether(m / 1000)
}

func (m Milliether) Uint64() uint64 {
	return uint64(m)
}

func (m Milliether) Float64() float64 {
	return float64(m)
}

type Ether float64

func (e Ether) Milliether() Milliether {
	return Milliether(e * 1000)
}

func (e Ether) Uint64() uint64 {
	return uint64(e)
}

func (e Ether) Float64() float64 {
	return float64(e)
}
