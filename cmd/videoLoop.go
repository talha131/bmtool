// Copyright © 2018 Talha Mansoor <talha131@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// videoLoopCmd represents the videoLoop command
var videoLoopCmd = &cobra.Command{
	Use:   "videoLoop",
	Short: "Concatenate same video multiple times to create a loop",
	Long: `Creates loop of a video by concatenating it multiple times.
Output format is mp4.

-c and -l are mutually exclusive. -c has precedence over -l.
`,
	Run: func(cmd *cobra.Command, args []string) {
		count, errC := cmd.Flags().GetUint16("count")
		length, errD := cmd.Flags().GetUint16("length")
		crossFade, _ := cmd.Flags().GetBool("withCrossFade")
		tDuration, _ := cmd.Flags().GetUint16("transitionDuration")

		if errC != nil && errD != nil {
			fmt.Fprint(os.Stderr, "Unable to find Count or Length. At least one is required")
			return
		}

		if count < 2 {
			fmt.Fprint(os.Stderr, "Loop count must be at least 2.")
			return
		}

		oPath := createOutputDirectory(cmd)
		shouldConcatCountTimes := length == 0 && errC == nil && count > 0
		shouldConcatToAchieveLength := !shouldConcatCountTimes && errD == nil && length > 0

		for _, e := range args {
			if isFileVideo(e) {
				if shouldConcatCountTimes {
					outputFileName := getOutputFileName(oPath, e, fmt.Sprintf("%s-%d", "loop", count))
					if !crossFade {
						createVideoLoopWithoutTransition(count, e, outputFileName)
					} else {
						createVideoLoopWithTransition(count, tDuration, e, outputFileName)
					}
				} else if shouldConcatToAchieveLength {
					count, err := getRequiredLoopCount(e, length)
					if err == nil {
						outputFileName := getOutputFileName(oPath, e, fmt.Sprintf("%s-%d", "length", length))
						if !crossFade {
							createVideoLoopWithoutTransition(count, e, outputFileName)
						}
					}
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(videoLoopCmd)

	videoLoopCmd.Flags().Uint16P("count", "c", 3, "Number of times to concatenate the video. Minimum 2.")
	videoLoopCmd.Flags().Uint16P("length", "l", 0, "Minimum minutes of the video")
	videoLoopCmd.Flags().BoolP("withCrossFade", "x", false, "Concatenate videos with cross fade transition")
	videoLoopCmd.Flags().Uint16P("transitionDuration", "t", 2, "Transition duration. Default is 2 seconds.")
	videoLoopCmd.Flags().StringP("outputDirectory", "o", "", "Output directory path. Default is current.")
}

func filterComplexWithCrossFade(count uint16, tDur uint16, length uint16) string {

	cf := ""
	cl := ""
	cfcl := ""
	var i uint16 = 1
	for ; i < count; i++ {
		cf = cf + fmt.Sprintf("[cf%d]", i)
		cl = cl + fmt.Sprintf("[cl%d]", i)
		cfcl = cfcl + fmt.Sprintf("[cf%d][cl%d]", i, i)
	}

	var a string

	// length = 15, tDur = 5
	a = a + fmt.Sprintf("[0:v]trim=start=0:end=%d,setpts=PTS-STARTPTS[clip1]; ", length-tDur)               // 0 - 10
	a = a + fmt.Sprintf("[0:v]trim=start=%d:end=%d,setpts=PTS-STARTPTS[clip2]; ", tDur, length-tDur)        // 5 - 10
	a = a + fmt.Sprintf("[0:v]trim=start=%d:end=%d,setpts=PTS-STARTPTS[clip3]; ", length-tDur, length)      // 10 - 15
	a = a + fmt.Sprintf("[0:v]trim=start=%d:end=%d,setpts=PTS-STARTPTS[fadeoutsrc]; ", length-tDur, length) // 10 - 15
	a = a + fmt.Sprintf("[0:v]trim=start=0:end=%d,setpts=PTS-STARTPTS[fadeinsrc]; ", tDur)                  // 0 - 5

	a = a + fmt.Sprintf("[fadeinsrc]format=pix_fmts=yuva420p, fade=t=in:st=0:d=%d:alpha=1[fadein]; ", tDur)
	a = a + fmt.Sprintf("[fadeoutsrc]format=pix_fmts=yuva420p, fade=t=out:st=0:d=%d:alpha=1[fadeout]; ", tDur)

	a = a + "[fadein]fifo[fadeinfifo]; "
	a = a + "[fadeout]fifo[fadeoutfifo]; "
	a = a + "[fadeoutfifo][fadeinfifo]overlay[crossfade]; "

	a = a + fmt.Sprintf("[crossfade] split=%d %s ; ", count-1, cf)
	a = a + fmt.Sprintf("[clip2] split=%d %s ; ", count-1, cl)

	a = a + "[clip1]" + cfcl + "[clip3]"
	// Final number of clips to concatenate is twice of count
	a = a + fmt.Sprintf("concat=n=%d:v=1[output]", count*2)

	return a
}

func createVideoLoopWithTransition(count uint16, tDur uint16, file string, outputFileName string) {
	l, err := getLength(file)
	if err != nil {
		return
	}

	length := uint16(l)

	if length < tDur {
		fmt.Fprint(os.Stderr, "Transition duration must be less than video length")
	}

	fc := filterComplexWithCrossFade(count, tDur, length)

	if v, _ := rootCmd.Flags().GetBool("verbose"); v {
		fmt.Printf("filter_complex is\n%s\n", fc)
	}

	cmd := exec.Command(app, "-hide_banner",
		"-i", file,
		"-an", "-filter_complex",
		fc,
		"-map", "[output]",
		outputFileName)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func getRequiredLoopCount(length int, requiredLength int, tDuration int) (int, error) {
	if requiredLength == 0 {
		fmt.Fprintf(os.Stderr, "Required length is invalid")
		return 0, errors.New("required duration is 0")
	}

	// totalLength = count x firstClip + lastClip
	// count = (totalLength - lastClip) / firstClip
	//
	// Here totalLength is the requiredLength.
	// lastClip is transitionDuration
	// firstClip is fileLength - transitionDuration
	// count = (requiredLength - tDuration) / (length - tDuration)
	numerator := float64(requiredLength - tDuration)
	denominator := float64(length - tDuration)
	requiredLoop := int(math.Ceil(numerator / denominator))

	if v, _ := rootCmd.Flags().GetBool("verbose"); v {
		fmt.Printf("Loop %d times\n", requiredLoop)
	}

	return requiredLoop, nil
}

func getOutputFileName(oPath string, f string, suffix string) string {
	fn := getFileNameWithoutExtension(f) + "_" + suffix + "." + "mp4"

	return filepath.Join(oPath, fn)
}

func createVideoLoopWithoutTransition(count uint16, e string, output string) {
	tmpFile, err := ioutil.TempFile(filepath.Dir(e), getFileNameWithoutExtension(e))
	if err != nil {
		log.Fatal(err)
	}

	defer os.Remove(tmpFile.Name()) // clean up

	p, _ := filepath.Abs(e)

	line := fmt.Sprintf("%s '%s'\n", "file", p)

	if v, _ := rootCmd.Flags().GetBool("verbose"); v {
		fmt.Printf("file is\n%v\n", line)
	}

	lineR := strings.Repeat(line, int(count))

	if _, err := tmpFile.WriteString(lineR); err != nil {
		log.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		log.Fatal(err)
	}

	runCommandVideoLoopWithoutTransition(tmpFile.Name(),
		output)
}

func runCommandVideoLoopWithoutTransition(file string, output string) {

	cmd := exec.Command(app, "-hide_banner",
		"-f", "concat",
		"-safe", "0",
		"-i", file,
		"-qscale:v", "0",
		output)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
