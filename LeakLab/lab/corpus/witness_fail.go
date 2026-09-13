package corpus

import "errors"

// witnessFail est le témoin de capture : son assertion est fausse par construction. Un détecteur
// qui ne voit pas cet échec, ou une capture qui perd ce message, rend ses verdicts non concluants.
func witnessFail() error {
	return errors.New("LEAKLAB-WITNESS : assertion fausse par construction")
}
