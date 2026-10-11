using System;

namespace MyApp
{
    public delegate void StatusChanged(string status);
    public delegate TResult Converter<TInput, TResult>(TInput input);

    public enum Direction { North, South, East, West }

    public interface IShape
    {
        double Area();
    }

    public record Point(double X, double Y);
    public record struct PointStruct(double X, double Y);

    public struct Vector2
    {
        public double X;
        public double Y;
    }

    public class Calculator : IShape
    {
        private int _value;

        public event StatusChanged OnStatusChanged;

        public int Value { get => _value; set => _value = value; }

        public Calculator(int initial) { _value = initial; }

        ~Calculator() { }

        public int Add(int a, int b) => a + b;

        public double Area() => _value;

        public int this[int index] => index;

        public static Calculator operator +(Calculator a, Calculator b)
            => new Calculator(a._value + b._value);
    }

    public partial class PartialClass
    {
        public void Method1() {}
    }
}
