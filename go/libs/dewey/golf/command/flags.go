package command

import "code.linenisgreat.com/chrest/go/libs/dewey/0/interfaces"

type CommandComponentReader interface {
	GetCLIFlags() []string
}

type CommandComponent interface {
	CommandComponentReader
	interfaces.CommandComponentWriter
}
