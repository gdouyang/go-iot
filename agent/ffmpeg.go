package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go-iot/provider/ffmpeg"
)

func GetAppPath() string {
	file, _ := exec.LookPath(os.Args[0])
	path, _ := filepath.Abs(file)
	index := strings.LastIndex(path, string(os.PathSeparator))

	return path[:index]
}

func GetfmgFrame(path string, dst string, name string) {
	arg := []string{"-y", "-i", path,
		"-rtsp_transport", "tcp",
		"-vframes", "1",
		"-ss", "2",
		"-s", fmt.Sprintf("%dx%d", 1920, 1080),
		"-f", "image2",
		fmt.Sprintf("%s%s.jpg", dst, name)}
	err := ffmpeg.RunAndClose(arg, nil)
	if err != nil {
		fmt.Println(err)
	}
	//	pp := ffmpeg.New(name, arg)
	//	err := pp.RunThenClose(ch)
	//	if err != nil {
	//		fmt.Println(err)
	//		return
	//	}
}
func GetfmgVideo(path string, dst string, name string, minu string) (ffmpeg.C, error) {
	arg := []string{
		"-i", path,
		"-c", "copy",
		"-f", "segment",
		"-segment_time", minu,
		"-segment_format", "flv",
		fmt.Sprintf("%s%s%%03d.flv", dst, name)}
	pp := ffmpeg.New(name, arg)
	err := pp.Run()
	if err != nil {
		fmt.Println(err)
		return pp, err
	}
	return pp, nil
}
func ClosefmgVideo(fc ffmpeg.C) error {
	fc.Close()
	return nil
}
func CloseAll(ac []ffmpeg.C) {
	for _, val := range ac {
		fmt.Println(val.Name)
		val.Close()
	}
}

func StreamLiveVideo(path string, dst string, name string, cam string) (ffmpeg.C, error) {
	arg := []string{
		"-i", path,
		"-c", "copy",
		"-f", "flv",
		"-an",
		fmt.Sprintf("%s%s", dst, name)}
	fmt.Printf("%s%s\n", dst, name)
	pp := ffmpeg.New(cam, arg)
	err := pp.Run()
	if err != nil {
		fmt.Println(err)
		return pp, err
	}
	return pp, nil
}
