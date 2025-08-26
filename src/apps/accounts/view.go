package accounts

import "strconv"

func (a Account) String() string {
	return a.Name + "(" + strconv.Itoa(a.Id) + ")"
}
