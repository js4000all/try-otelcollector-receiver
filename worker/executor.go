package worker

import (
	"context"
)

type TargetID string

// Command は、Worker が扱う「処理要求」。
// WorkerManager / Worker は中身を解釈しない。
type Command interface{}

// Result も WorkerManager / Worker は解釈しない。
// 実際には後で型を詰めてもよい。
type Result any

// Executor が Command の実際の処理を担当する。
// TCP / Host Link の知識はここから下に閉じ込める。
type Executor interface {
	Execute(ctx context.Context, command Command) (Result, error)
}

// Request は Worker のキューに流す内部メッセージ。
type Request struct {
	Context context.Context
	Command Command
	Reply   chan<- Response
}

type Response struct {
	Result Result
	Err    error
}
