import 'package:godwit/src/util/csv.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('parses quotes, escaped quotes, CRLF and a byte order mark', () {
    const text = '\uFEFFname,permissions\r\n"Desk, oak",view\r\n"He said ""hi""","view, edit"\n';
    expect(parseCsv(text), [
      ['name', 'permissions'],
      ['Desk, oak', 'view'],
      ['He said "hi"', 'view, edit'],
    ]);
  });

  test('handles a missing final newline and maxRows', () {
    expect(parseCsv('a,b\n1,2'), [
      ['a', 'b'],
      ['1', '2'],
    ]);
    expect(parseCsv('a\n1\n2\n3\n', maxRows: 2), [
      ['a'],
      ['1'],
    ]);
  });
}
