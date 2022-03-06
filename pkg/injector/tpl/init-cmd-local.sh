#
# The NEU License
#
# Copyright (c) 2022.  flomesh.io
#
# Permission is hereby granted, free of charge, to any person obtaining a copy of
# this software and associated documentation files (the "Software"), to deal in
# the Software without restriction, including without limitation the rights to
# use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
# of the Software, and to permit persons to whom the Software is furnished to do
# so, subject to the following conditions:
#
# (1)The above copyright notice and this permission notice shall be included in
# all copies or substantial portions of the Software.
#
# (2)If the software or part of the code will be directly used or used as a
# component for commercial purposes, including but not limited to: public cloud
#  services, hosting services, and/or commercial software, the logo as following
#  shall be displayed in the eye-catching position of the introduction materials
# of the relevant commercial services or products (such as website, product
# publicity print), and the logo shall be linked or text marked with the
# following URL.
#
# LOGO : http://flomesh.cn/assets/flomesh-logo.png
# URL : https://github.com/flomesh-io
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.
#

mkdir -p -v $(_SIDECAR_CONFIG_PATH_)
cp -fv %s/* $(_SIDECAR_CONFIG_PATH_)
ls -l $(_SIDECAR_CONFIG_PATH_)