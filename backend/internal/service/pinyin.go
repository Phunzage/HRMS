package service

import "strings"

// toPinyin 将姓名转成拼音（无音调，小写），用于生成默认用户名。
// 未收录的汉字直接跳过；若全部未收录则返回空串，由上层用 emp+序号 兜底。
func toPinyin(s string) string {
	var b strings.Builder
	for _, r := range s {
		if p, ok := pinyinMap[r]; ok {
			b.WriteString(p)
		}
	}
	return b.String()
}

// pinyinMap 常用汉字 → 拼音（无音调），覆盖常见姓氏与名字用字
var pinyinMap = map[rune]string{
	// 常见姓氏
	'王': "wang", '李': "li", '张': "zhang", '刘': "liu", '陈': "chen",
	'杨': "yang", '黄': "huang", '赵': "zhao", '吴': "wu", '周': "zhou",
	'徐': "xu", '孙': "sun", '马': "ma", '朱': "zhu", '胡': "hu",
	'郭': "guo", '何': "he", '高': "gao", '林': "lin", '罗': "luo",
	'郑': "zheng", '梁': "liang", '谢': "xie", '宋': "song", '唐': "tang",
	'许': "xu", '韩': "han", '冯': "feng", '邓': "deng", '曹': "cao",
	'彭': "peng", '曾': "zeng", '萧': "xiao", '田': "tian", '董': "dong",
	'潘': "pan", '袁': "yuan", '蒋': "jiang", '蔡': "cai", '余': "yu",
	'杜': "du", '叶': "ye", '程': "cheng", '苏': "su", '魏': "wei",
	'吕': "lv", '丁': "ding", '任': "ren", '沈': "shen", '姚': "yao",
	'卢': "lu", '姜': "jiang", '崔': "cui", '钟': "zhong", '谭': "tan",
	'陆': "lu", '汪': "wang", '范': "fan", '金': "jin", '石': "shi",
	'廖': "liao", '贾': "jia", '夏': "xia", '韦': "wei", '付': "fu",
	'方': "fang", '白': "bai", '邹': "zou", '孟': "meng", '熊': "xiong",
	'秦': "qin", '邱': "qiu", '江': "jiang", '尹': "yin", '薛': "xue",
	'闫': "yan", '段': "duan", '雷': "lei", '侯': "hou", '龙': "long",
	'史': "shi", '陶': "tao", '黎': "li", '贺': "he", '顾': "gu",
	'毛': "mao", '郝': "hao", '龚': "gong", '邵': "shao", '万': "wan",
	'钱': "qian", '严': "yan", '覃': "qin", '武': "wu", '戴': "dai",
	'莫': "mo", '孔': "kong", '向': "xiang", '汤': "tang", '欧': "ou",
	'蓝': "lan", '项': "xiang",
	// 常见名字用字
	'伟': "wei", '芳': "fang", '娜': "na", '敏': "min", '静': "jing",
	'丽': "li", '强': "qiang", '磊': "lei", '军': "jun", '洋': "yang",
	'勇': "yong", '艳': "yan", '杰': "jie", '娟': "juan", '涛': "tao",
	'明': "ming", '超': "chao", '秀': "xiu", '霞': "xia", '平': "ping",
	'刚': "gang", '桂': "gui", '英': "ying", '华': "hua", '玉': "yu",
	'国': "guo", '文': "wen", '辉': "hui", '红': "hong", '梅': "mei",
	'斌': "bin", '鹏': "peng", '珍': "zhen", '荣': "rong", '成': "cheng",
	'兰': "lan", '雪': "xue", '海': "hai", '春': "chun", '丹': "dan",
	'燕': "yan", '慧': "hui", '玲': "ling", '婷': "ting", '雨': "yu",
	'欣': "xin", '晨': "chen", '宇': "yu", '轩': "xuan", '浩': "hao",
	'然': "ran", '博': "bo", '涵': "han", '睿': "rui", '泽': "ze",
	'远': "yuan", '乐': "le", '佳': "jia", '琪': "qi", '瑶': "yao",
	'若': "ruo", '思': "si", '俊': "jun", '凯': "kai",
	'瑞': "rui", '天': "tian", '翼': "yi", '子': "zi", '书': "shu",
	'志': "zhi", '建': "jian", '永': "yong", '光': "guang", '峰': "feng",
}
