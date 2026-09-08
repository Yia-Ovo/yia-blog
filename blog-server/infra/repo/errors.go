package repo

import (
	"errors"

	domainrepo "blog-server/domain/repo"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

const mysqlDuplicateEntry = 1062

// translateDuplicateKey 在持久化边界识别唯一约束冲突，兼容原生和 GORM 翻译后的错误。
func translateDuplicateKey(err error) error {
	var mysqlErr *mysql.MySQLError
	if errors.Is(err, gorm.ErrDuplicatedKey) || errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlDuplicateEntry {
		return errors.Join(domainrepo.ErrDuplicateKey, err)
	}
	return err
}
