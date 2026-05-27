package recipe

type HTML struct{}

func (HTML) Execute(c *Context) (*Result, error) {
	return &Result{
		Content:  []byte(c.HTML),
		MimeType: "text/html; charset=utf-8",
		FileName: "report.html",
	}, nil
}

type Text struct{}

func (Text) Execute(c *Context) (*Result, error) {
	return &Result{
		Content:  []byte(c.HTML),
		MimeType: "text/plain; charset=utf-8",
		FileName: "report.txt",
	}, nil
}
