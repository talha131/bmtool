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
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// videoLoopCmd represents the videoLoop command
var videoLoopCmd = &cobra.Command{
	Use:   "videoLoop",
	Short: "Concatenate same video multiple times to create a loop",
	Long: `Creates loop of a video by contactenating it multiple times.
It picks only the video stream and discards audio stream.
Output format is mp4.

-c and -d are mutually exclusive.
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("videoLoop called")
	},
}

func init() {
	rootCmd.AddCommand(videoLoopCmd)

	videoLoopCmd.Flags().Uint16P("count", "c", 3, "Number of times to concatenate the video")
	videoLoopCmd.Flags().Uint16P("duration", "d", 0, "Minimum minutes of the video")
	videoLoopCmd.Flags().StringP("outputDirectory", "o", "", "Output directory path. Default is current.")
}
