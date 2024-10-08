// SPDX-License-Identifier: GPL-3.0-or-later

#include "sanitizers-functions.h"

static unsigned char functions_allowed_chars[256] = {
        [0] = '\0', [1] = ' ', [2] = ' ', [3] = ' ', [4] = ' ', [5] = ' ', [6] = ' ', [7] = ' ', [8] = ' ',

        // control characters to be treated as spaces
        ['\t'] = ' ', ['\n'] = ' ', ['\v'] = ' ', ['\f'] = ' ', ['\r'] = ' ',

        [14] = ' ', [15] = ' ', [16] = ' ', [17] = ' ', [18] = ' ', [19] = ' ', [20] = ' ', [21] = ' ',
        [22] = ' ', [23] = ' ', [24] = ' ', [25] = ' ', [26] = ' ', [27] = ' ', [28] = ' ', [29] = ' ',
        [30] = ' ', [31] = ' ',

        // symbols
        [' '] = ' ', ['!'] = '!', ['"'] = '\'', ['#'] = '#', ['$'] = '$', ['%'] = '%', ['&'] = '&', ['\''] = '\'',
        ['('] = '(', [')'] = ')', ['*'] = '*', ['+'] = '+', [','] = ',', ['-'] = '-', ['.'] = '.', ['/'] = '/',

        // numbers
        ['0'] = '0', ['1'] = '1', ['2'] = '2', ['3'] = '3', ['4'] = '4', ['5'] = '5', ['6'] = '6', ['7'] = '7',
        ['8'] = '8', ['9'] = '9',

        // symbols
        [':'] = ':', [';'] = ';', ['<'] = '<', ['='] = '=', ['>'] = '>', ['?'] = '?', ['@'] = '@',

        // capitals
        ['A'] = 'A', ['B'] = 'B', ['C'] = 'C', ['D'] = 'D', ['E'] = 'E', ['F'] = 'F', ['G'] = 'G', ['H'] = 'H',
        ['I'] = 'I', ['J'] = 'J', ['K'] = 'K', ['L'] = 'L', ['M'] = 'M', ['N'] = 'N', ['O'] = 'O', ['P'] = 'P',
        ['Q'] = 'Q', ['R'] = 'R', ['S'] = 'S', ['T'] = 'T', ['U'] = 'U', ['V'] = 'V', ['W'] = 'W', ['X'] = 'X',
        ['Y'] = 'Y', ['Z'] = 'Z',

        // symbols
        ['['] = '[', ['\\'] = '\\', [']'] = ']', ['^'] = '^', ['_'] = '_', ['`'] = '`',

        // lower
        ['a'] = 'a', ['b'] = 'b', ['c'] = 'c', ['d'] = 'd', ['e'] = 'e', ['f'] = 'f', ['g'] = 'g', ['h'] = 'h',
        ['i'] = 'i', ['j'] = 'j', ['k'] = 'k', ['l'] = 'l', ['m'] = 'm', ['n'] = 'n', ['o'] = 'o', ['p'] = 'p',
        ['q'] = 'q', ['r'] = 'r', ['s'] = 's', ['t'] = 't', ['u'] = 'u', ['v'] = 'v', ['w'] = 'w', ['x'] = 'x',
        ['y'] = 'y', ['z'] = 'z',

        // symbols
        ['{'] = '{', ['|'] = '|', ['}'] = '}', ['~'] = '~',

        // rest
        [127] = ' ', [128] = ' ', [129] = ' ', [130] = ' ', [131] = ' ', [132] = ' ', [133] = ' ', [134] = ' ',
        [135] = ' ', [136] = ' ', [137] = ' ', [138] = ' ', [139] = ' ', [140] = ' ', [141] = ' ', [142] = ' ',
        [143] = ' ', [144] = ' ', [145] = ' ', [146] = ' ', [147] = ' ', [148] = ' ', [149] = ' ', [150] = ' ',
        [151] = ' ', [152] = ' ', [153] = ' ', [154] = ' ', [155] = ' ', [156] = ' ', [157] = ' ', [158] = ' ',
        [159] = ' ', [160] = ' ', [161] = ' ', [162] = ' ', [163] = ' ', [164] = ' ', [165] = ' ', [166] = ' ',
        [167] = ' ', [168] = ' ', [169] = ' ', [170] = ' ', [171] = ' ', [172] = ' ', [173] = ' ', [174] = ' ',
        [175] = ' ', [176] = ' ', [177] = ' ', [178] = ' ', [179] = ' ', [180] = ' ', [181] = ' ', [182] = ' ',
        [183] = ' ', [184] = ' ', [185] = ' ', [186] = ' ', [187] = ' ', [188] = ' ', [189] = ' ', [190] = ' ',
        [191] = ' ', [192] = ' ', [193] = ' ', [194] = ' ', [195] = ' ', [196] = ' ', [197] = ' ', [198] = ' ',
        [199] = ' ', [200] = ' ', [201] = ' ', [202] = ' ', [203] = ' ', [204] = ' ', [205] = ' ', [206] = ' ',
        [207] = ' ', [208] = ' ', [209] = ' ', [210] = ' ', [211] = ' ', [212] = ' ', [213] = ' ', [214] = ' ',
        [215] = ' ', [216] = ' ', [217] = ' ', [218] = ' ', [219] = ' ', [220] = ' ', [221] = ' ', [222] = ' ',
        [223] = ' ', [224] = ' ', [225] = ' ', [226] = ' ', [227] = ' ', [228] = ' ', [229] = ' ', [230] = ' ',
        [231] = ' ', [232] = ' ', [233] = ' ', [234] = ' ', [235] = ' ', [236] = ' ', [237] = ' ', [238] = ' ',
        [239] = ' ', [240] = ' ', [241] = ' ', [242] = ' ', [243] = ' ', [244] = ' ', [245] = ' ', [246] = ' ',
        [247] = ' ', [248] = ' ', [249] = ' ', [250] = ' ', [251] = ' ', [252] = ' ', [253] = ' ', [254] = ' ',
        [255] = ' '
};

size_t rrd_functions_sanitize(char *dst, const char *src, size_t dst_len) {
    return text_sanitize((unsigned char *)dst, (const unsigned char *)src, dst_len,
                         functions_allowed_chars, true, "", NULL);
}

