package page

// Param 分页参数
type Param struct {
	PageSize int32
	Current  int32
}

// GetPageSize 获取每页数量
func (p *Param) GetPageSize() int32 {
	if p.PageSize <= 0 {
		p.PageSize = 10
	}
	return p.PageSize
}

// GetCurrent 获取页码
func (p *Param) GetCurrent() int32 {
	if p.Current <= 0 {
		p.Current = 1
	}
	return p.Current
}
