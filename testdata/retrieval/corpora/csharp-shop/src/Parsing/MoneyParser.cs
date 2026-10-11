using System;
using System.Globalization;
using System.Text;

namespace Shop.Parsing
{
    public static class MoneyParser
    {
        public static decimal Parse(string text)
        {
            ArgumentNullException.ThrowIfNull(text);
            return Parse(text.AsSpan(), NumberStyles.Currency);
        }

        public static decimal Parse(ReadOnlySpan<char> text, NumberStyles styles)
        {
            var trimmed = text.Trim();
            if (!decimal.TryParse(trimmed, styles, CultureInfo.InvariantCulture, out var value))
            {
                throw new FormatException($"Invalid amount: '{trimmed.ToString()}'");
            }
            return value;
        }

        public static decimal Parse(ReadOnlySpan<byte> utf8)
        {
            Span<char> buffer = stackalloc char[Encoding.UTF8.GetMaxCharCount(utf8.Length)];
            var written = Encoding.UTF8.GetChars(utf8, buffer);
            return Parse(buffer[..written], NumberStyles.Currency);
        }
    }
}
