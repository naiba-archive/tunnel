/*
 * Copyright (c) 2018, 奶爸<1@5.nu>
 * All rights reserved.
 */

package manager

const CodeReg = 1001
const CodeRegSuccess = 1002
const CodeLogin = 1003
const CodeLoginSuccess = 1004
const CodeNeedLogin = 1005
const CodeUpdateForwardCTable = 2001
const CodeError = 5000

type Pocket struct {
	Code int
	Data string
}
