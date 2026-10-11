public class Calculator {
    public static final int MAX = 100;
    private String name;

    public Calculator() {}

    public int add(int a, int b) {
        return a + b;
    }

    class InnerClass {
        void innerMethod() {}
    }
}

interface Computable {
    int compute();
}

enum Status {
    ACTIVE, INACTIVE;

    public String label() { return name().toLowerCase(); }
}

@interface MyAnnotation {
    String value();
}

record Point(int x, int y) {
    public double distance() {
        return Math.sqrt(x * x + y * y);
    }
}

// class CommentedClass {}
