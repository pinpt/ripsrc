// +build no_license

// go-license-detector slows down runtime by 5s initializing something
// use no_license tag when testing unrelated components that does not include that package
package fileinfo

func detect(filename string, buf []byte) (*License, error) {
	return nil, nil
}
