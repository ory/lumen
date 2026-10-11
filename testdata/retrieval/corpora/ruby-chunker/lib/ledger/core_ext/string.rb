# frozen_string_literal: true

class String
  def to_cents
    whole, fraction = delete(",").split(".")
    (whole.to_i * 100) + fraction.to_s.ljust(2, "0")[0, 2].to_i
  end
end
