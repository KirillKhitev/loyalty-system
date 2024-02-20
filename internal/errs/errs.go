package errs

import "errors"

var (
	ErrOrderExistsByThisUser  = errors.New("this order also exists")
	ErrOrderExistsByOtherUser = errors.New("this order also exists by other User")
)
