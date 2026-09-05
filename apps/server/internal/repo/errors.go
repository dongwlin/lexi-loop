package repo

// TODO: 定义 Repo 可识别错误，例如：
//
//	var (
//	    ErrNotFound          = errors.New("repo: not found")
//	    ErrConflict          = errors.New("repo: concurrent update conflict")
//	    ErrAlreadyExists     = errors.New("repo: unique constraint violation")
//	    ErrReferenceViolation = errors.New("repo: foreign key violation")
//	)
//
// 通过 *pgconn.PgError 的 SQLSTATE 判断约束类错误，不比较驱动错误文本。
