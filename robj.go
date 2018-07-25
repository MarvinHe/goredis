package go_redis

type robji interface {
	UpdateLRU() error
	GetType() uint8
}

type robjBase struct {
	typeEncode  byte
	lruFlag     int16
	refCount    int16
}

func (o robjBase) GetType() uint8 {
	// decide type by typeEncode
	return (o.typeEncode >> 4) & 0xF
}

func (o robjBase) UpdateLRU() error {
	o.lruFlag = 1
	return nil
}
