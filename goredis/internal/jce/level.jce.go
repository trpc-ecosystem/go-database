package jce

import (
	"trpc.group/trpc-go/jce"
)

// GetLevelReq struct implement
type GetLevelReq struct {
	Uin int64 `json:"uin"`
}

// ResetDefault reset default
func (st *GetLevelReq) ResetDefault() {
}

// ReadFrom reads  from _is and put into struct.
func (st *GetLevelReq) ReadFrom(_is *jce.Reader) error {
	var err error
	var length int32
	var have bool
	var ty byte
	st.ResetDefault()
	err = _is.Read_int64(&st.Uin, 0, false)
	if err != nil {
		return err
	}
	_ = length
	_ = have
	_ = ty
	return nil
}

// WriteTo encode struct to buffer
func (st *GetLevelReq) WriteTo(_os *jce.Buffer) error {
	return _os.Write_int64(st.Uin, 0)
}
