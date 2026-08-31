### splitFunc
- parser 参数说明

| 方法 | 说明 | 参数 | 返回值 |
| --- | --- | ---- | ---- |
| AddHandler | 添加报文处理器 | function | - |
| AppendResult | 追加结果 | data | - |
| Complete | 完成解析 | - | - |
| Fixed | 按固定长度读取数据 | size: number | - |
| Delimited | 按分隔符读取数据 | word: string | - |

- 样例1
> 数据格式为`aasss 2023-01-18 22:15:25_\n`, 解析后的数据为`aasss 2023-01-18 22:15:25_`
```javascript
function splitFunc(parser) {
  // 2. 拆分后的数据进入Handler
  parser.AddHandler(function(data) {
    // 3. 把数据追加到结果
    parser.AppendResult(data);
    // 4. 完成解析
    parser.Complete();
  });
  // 1. 按\n来拆分数据
  parser.Delimited("\\n")
}
```
- 样例2
> 数据格式为`aasss 2023-01-18 22:15:25_\n`, 解析后的数据为`2023-01-18 22:15:25_`
```javascript
function splitFunc(parser) {
  // 2. 拆分后的数据进入Handler, data1由于没有追加到结果中所以丢弃了
  parser.AddHandler(function(data1) {
    // 4. 拆分后的数据进入Handler
    parser.AddHandler(function(data2) {
      // 5. 把数据追加到结果
      parser.AppendResult(data2);
      // 6. 完成解析
      parser.Complete();
    });
    // 3. 再使用\n来拆分数据
    parser.Delimited("\\n")
  });
  // 1. 使用空格来拆分数据
  parser.Delimited(" ");
}
```
- 样例3
> 数据格式为`aasss 2023-01-18 22:15:25_\n`, 解析后的数据为`2023-01-18 22:15:25_\n`
```javascript
function splitFunc(parser) {
  // 2. 读取后进入Handler, data1由于没有追加到结果中所以丢弃了
  parser.AddHandler(function(data1) {
    // 4. 21个长度的数据进入Handler
    parser.AddHandler(function(data2) {
      // 5. 把21个长度的数据放到结果中
      parser.AppendResult(data2);
      // 6. 完成解析
      parser.Complete();
    });
    // 3. 再读取21个长度的数据
    parser.Fixed(21);
  });
  // 1. 先读取6个长度的数据
  parser.Fixed(6) 
}
```